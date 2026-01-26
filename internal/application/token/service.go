package token

//
//import (
//	"context"
//	"sync"
//	"time"
//
//	"testtask/internal/Provider/token"
//)
//
//// Service manages token data updates from CoinGecko
//type Service struct {
//	provider token.CoinGeckoDataProvider
//	mu       sync.RWMutex
//	running  bool
//	stopCh   chan struct{}
//	wg       sync.WaitGroup
//}
//
//// NewService creates a new token service
//func NewService(provider token.CoinGeckoDataProvider) *Service {
//	return &Service{
//		provider: provider,
//		stopCh:   make(chan struct{}),
//	}
//}
//
//// Start begins the async update process that runs every 5 minutes
//func (s *Service) Start(ctx context.Context) error {
//	s.mu.Lock()
//	if s.running {
//		s.mu.Unlock()
//		return nil
//	}
//	s.running = true
//	s.mu.Unlock()
//
//	// Perform initial load
//	if err := s.updateData(ctx); err != nil {
//		// Log error but continue
//	}
//
//	// Start background goroutine for periodic updates
//	s.wg.Add(1)
//	go s.updateLoop(ctx)
//
//	return nil
//}
//
//// Stop stops the async update process
//func (s *Service) Stop() {
//	s.mu.Lock()
//	if !s.running {
//		s.mu.Unlock()
//		return
//	}
//	s.running = false
//	s.mu.Unlock()
//
//	close(s.stopCh)
//	s.wg.Wait()
//}
//
//// updateLoop runs the periodic update loop
//func (s *Service) updateLoop(ctx context.Context) {
//	defer s.wg.Done()
//
//	ticker := time.NewTicker(5 * time.Minute)
//	defer ticker.Stop()
//
//	for {
//		select {
//		case <-ticker.C:
//			if err := s.updateData(ctx); err != nil {
//				// Log error but continue
//			}
//		case <-s.stopCh:
//			return
//		case <-ctx.Done():
//			return
//		}
//	}
//}
//
//// updateData performs a full update of all token data
//func (s *Service) updateData(ctx context.Context) error {
//	// Load coins list
//	if _, err := s.provider.LoadCoinsList(ctx); err != nil {
//		return err
//	}
//
//	// Load platforms
//	if _, err := s.provider.LoadPlatforms(ctx); err != nil {
//		return err
//	}
//
//	// Update coins with decimals from token lists
//	if err := s.provider.UpdateCoinsWithDecimals(ctx); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//// IsRunning returns whether the service is currently running
//func (s *Service) IsRunning() bool {
//	s.mu.RLock()
//	defer s.mu.RUnlock()
//	return s.running
//}
