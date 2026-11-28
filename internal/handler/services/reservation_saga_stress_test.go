package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"strings"
	"sync"
	"testing"
)

func TestReservationAndOrderSaga_2500StressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(20261128))

	// 1. 1,000 predictable & unpredictable cases for validateCreateReservationInput
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		req := dto.CreateReservationRequest{
			ProductID: fmt.Sprintf("prod-%d", i),
			Quantity:  rng.Intn(600) - 50,
			HoldHours: rng.Intn(40) - 5,
			Notes:     strings.Repeat("n", rng.Intn(1300)),
		}

		// Inject unpredictable blanks
		switch i % 10 {
		case 0:
			userID = "   "
		case 1:
			req.ProductID = ""
		case 2:
			req.Quantity = 0 // should normalize to 1
		case 3:
			req.Quantity = -1
		case 4:
			req.Quantity = 501
		}

		err := validateCreateReservationInput(userID, &req)
		shouldFail := strings.TrimSpace(userID) == "" ||
			strings.TrimSpace(req.ProductID) == "" ||
			req.Quantity < 0 ||
			req.Quantity > 500

		if shouldFail && err == nil {
			t.Fatalf("case %d: expected error for invalid reservation input (user=%q prod=%q qty=%d)", i, userID, req.ProductID, req.Quantity)
		}
		if !shouldFail && err != nil {
			t.Fatalf("case %d: unexpected error for valid reservation input: %v", i, err)
		}
		if err == nil {
			if req.Quantity < 1 || req.Quantity > 500 {
				t.Fatalf("case %d: normalized quantity out of bounds: %d", i, req.Quantity)
			}
			if len(req.Notes) > 1000 {
				t.Fatalf("case %d: notes exceeded 1000 chars: %d", i, len(req.Notes))
			}
		}
	}

	// 2. 500 cases for normalizeVerifyCode & pickup/reservation generators
	badCodes := []string{"", "   ", "#", "  #  "}
	for i := 0; i < 500; i++ {
		if i < 100 {
			c := badCodes[i%len(badCodes)]
			if _, _, err := normalizeVerifyCode(c, ""); err == nil {
				t.Fatalf("expected error for blank/hash-only verify code %q", c)
			}
			continue
		}

		pickup := generatePickupCode()
		resNum := generateReservationNumber()
		if len(pickup) != 6 {
			t.Fatalf("expected 6-digit pickup code, got %q", pickup)
		}
		if !strings.HasPrefix(resNum, "HLD-") {
			t.Fatalf("expected HLD- prefix on reservation number, got %q", resNum)
		}

		var pIn, rIn string
		if i%2 == 0 {
			pIn = "#" + pickup
		} else {
			rIn = resNum
		}
		cOut, rOut, err := normalizeVerifyCode(pIn, rIn)
		if err != nil {
			t.Fatalf("unexpected error normalizing valid codes (%q, %q): %v", pIn, rIn, err)
		}
		if cOut == "" || rOut == "" {
			t.Fatalf("expected both normalized outputs non-empty, got (%q, %q)", cOut, rOut)
		}
	}

	// 3. 500 cases for validateOrderCheckoutSagaInput
	for i := 0; i < 500; i++ {
		in := OrderCheckoutSagaInput{
			ShopOwnerUserID:     fmt.Sprintf("owner-%d", i),
			Items:               []dto.POSSaleItemRequest{{ProductID: "p1", Quantity: 1}},
			DiscountAmount:      float64(rng.Intn(200)) - 20.0,
			RedeemLoyaltyPoints: rng.Intn(50) - 5,
		}
		switch i % 7 {
		case 0:
			in.ShopOwnerUserID = "  "
		case 1:
			in.Items = nil
		case 2:
			in.DiscountAmount = math.NaN()
		case 3:
			in.DiscountAmount = math.Inf(1)
		}
		err := validateOrderCheckoutSagaInput(in)
		expectErr := strings.TrimSpace(in.ShopOwnerUserID) == "" ||
			len(in.Items) == 0 ||
			math.IsNaN(in.DiscountAmount) ||
			math.IsInf(in.DiscountAmount, 0) ||
			in.DiscountAmount < 0 ||
			in.RedeemLoyaltyPoints < 0
		if expectErr && err == nil {
			t.Fatalf("saga validation case %d: expected error", i)
		}
		if !expectErr && err != nil {
			t.Fatalf("saga validation case %d: unexpected error: %v", i, err)
		}
	}

	// 4. 500 concurrent & sequential SagaCoordinator execution/compensation stress tests
	var wg sync.WaitGroup
	errCh := make(chan error, 500)
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			coord := newSagaCoordinator()
			failAt := idx % 4 // 0=succeed, 1..3=fail at step
			compCount := 0

			for s := 1; s <= 3; s++ {
				stepNum := s
				coord.AddStep(
					fmt.Sprintf("step-%d", stepNum),
					func(ctx context.Context) error {
						if failAt == stepNum {
							return errors.New("boom")
						}
						return nil
					},
					func(ctx context.Context) error {
						compCount++
						return nil
					},
				)
			}

			err := coord.Run(context.Background())
			if failAt == 0 {
				if err != nil || compCount != 0 {
					errCh <- fmt.Errorf("idx %d: expected clean run, got err=%v comp=%d", idx, err, compCount)
				}
			} else {
				wantComp := failAt - 1
				if err == nil || compCount != wantComp {
					errCh <- fmt.Errorf("idx %d: expected err and %d compensations, got err=%v comp=%d", idx, wantComp, err, compCount)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
