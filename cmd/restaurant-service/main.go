package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	headerAPIKey         = "X-Restaurant-API-Key"
	defaultPlatformURL   = "http://localhost:8080"
	defaultPollInterval  = 5 * time.Second
	httpClientTimeout    = 10 * time.Second
	delayPendingAccepted = 3 * time.Second
	delayAcceptedPrep    = 5 * time.Second
	delayPrepReady       = 10 * time.Second
)

// subOrderDTO — минимальное подмножество SubOrder для polling.
// Полные типы лежат в internal/api/generated.go, но здесь достаточно
// id/status/order_id чтобы не тянуть echo-зависимости в имитатор.
type subOrderDTO struct {
	ID      int    `json:"id"`
	OrderID int    `json:"order_id"`
	Status  string `json:"status"`
}

type statusUpdateRequest struct {
	Status string `json:"status"`
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getPollInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("POLL_INTERVAL"))
	if raw == "" {
		return defaultPollInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("WARN: invalid POLL_INTERVAL=%q, using default %s", raw, defaultPollInterval)
		return defaultPollInterval
	}
	return d
}

// sleepCtx спит d с учётом отмены контекста.
// Возвращает false если контекст отменён раньше.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func fetchOrders(client *http.Client, baseURL, apiKey string) ([]subOrderDTO, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/api/v1/restaurant/orders"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerAPIKey, apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{method: http.MethodGet, url: url, code: resp.StatusCode}
	}

	var orders []subOrderDTO
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, err
	}
	return orders, nil
}

type httpError struct {
	method string
	url    string
	code   int
}

func (e *httpError) Error() string {
	return "request " + e.method + " " + e.url + ": unexpected status " + http.StatusText(e.code) + " (" + strconv.Itoa(e.code) + ")"
}

func updateStatus(client *http.Client, baseURL, apiKey string, subOrderID int, status string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/api/v1/restaurant/sub-orders/" + strconv.Itoa(subOrderID) + "/status"
	body, err := json.Marshal(statusUpdateRequest{Status: status})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set(headerAPIKey, apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &httpError{method: http.MethodPatch, url: url, code: resp.StatusCode}
	}
	return nil
}

// processSubOrder отрабатывает цепочку pending->accepted->preparing->ready
// с задержками 3s/5s/10s. При ошибке PATCH цепочка прерывается.
func processSubOrder(ctx context.Context, client *http.Client, baseURL, apiKey, restaurantID string, wg *sync.WaitGroup, subOrderID, orderID int) {
	defer wg.Done()

	steps := []struct {
		status string
		delay  time.Duration
	}{
		{"accepted", delayPendingAccepted},
		{"preparing", delayAcceptedPrep},
		{"ready", delayPrepReady},
	}

	log.Printf("[restaurant %s] start processing sub-order id=%d order_id=%d", restaurantID, subOrderID, orderID)
	for _, s := range steps {
		if !sleepCtx(ctx, s.delay) {
			log.Printf("[restaurant %s] sub-order id=%d: shutdown during wait for status=%s", restaurantID, subOrderID, s.status)
			return
		}
		if err := updateStatus(client, baseURL, apiKey, subOrderID, s.status); err != nil {
			log.Printf("[restaurant %s] sub-order id=%d: PATCH status=%s failed: %v", restaurantID, subOrderID, s.status, err)
			return
		}
		log.Printf("[restaurant %s] sub-order id=%d: status -> %s", restaurantID, subOrderID, s.status)
	}
	log.Printf("[restaurant %s] sub-order id=%d: done (ready)", restaurantID, subOrderID)
}

func pollOnce(ctx context.Context, client *http.Client, baseURL, apiKey, restaurantID string, seen map[int]struct{}, mu *sync.Mutex, wg *sync.WaitGroup) {
	orders, err := fetchOrders(client, baseURL, apiKey)
	if err != nil {
		// Ошибка сети/статус — логируем и ждём следующего тика (retry на следующем тике).
		log.Printf("[restaurant %s] poll failed: %v (retry on next tick)", restaurantID, err)
		return
	}
	log.Printf("[restaurant %s] poll ok: fetched %d sub-orders", restaurantID, len(orders))

	for _, o := range orders {
		if o.Status != "pending" {
			continue
		}
		mu.Lock()
		if _, ok := seen[o.ID]; ok {
			mu.Unlock()
			continue
		}
		seen[o.ID] = struct{}{}
		mu.Unlock()

		log.Printf("[restaurant %s] new pending sub-order id=%d order_id=%d", restaurantID, o.ID, o.OrderID)
		wg.Add(1)
		go processSubOrder(ctx, client, baseURL, apiKey, restaurantID, wg, o.ID, o.OrderID)
	}
}

func main() {
	platformURL := getenv("PLATFORM_API_URL", defaultPlatformURL)
	apiKey := strings.TrimSpace(os.Getenv("RESTAURANT_API_KEY"))
	restaurantID := getenv("RESTAURANT_ID", "unknown")
	pollInterval := getPollInterval()

	if apiKey == "" {
		log.Fatal("RESTAURANT_API_KEY is required")
	}

	log.Printf("[restaurant %s] starting: platform=%s poll_interval=%s", restaurantID, platformURL, pollInterval)

	client := &http.Client{Timeout: httpClientTimeout}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		log.Printf("[restaurant %s] received signal %s, shutting down", restaurantID, sig)
		cancel()
	}()

	var (
		seen = make(map[int]struct{})
		mu   sync.Mutex
		wg   sync.WaitGroup
	)

	// Первый опрос сразу, затем по тикеру.
	pollOnce(ctx, client, platformURL, apiKey, restaurantID, seen, &mu, &wg)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[restaurant %s] stopping polling loop, waiting for in-flight sub-orders", restaurantID)
			wg.Wait()
			log.Printf("[restaurant %s] stopped", restaurantID)
			return
		case <-ticker.C:
			pollOnce(ctx, client, platformURL, apiKey, restaurantID, seen, &mu, &wg)
		}
	}
}
