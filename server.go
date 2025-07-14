package gocap

import (
	"encoding/json"
	"net/http"
)

// HandleCreateChallenge 创建质询 Http Handle 封装
func (c *Cap) HandleCreateChallenge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.limiter[0] != nil && !c.limiter[0].Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		challenge, err := c.CreateChallenge(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(challenge)
	}
}

// HandleRedeemChallenge 工作量证明兑换验证令牌 Http Handle 封装
func (c *Cap) HandleRedeemChallenge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.limiter[1] != nil && !c.limiter[1].Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var params VerificationParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var resp VerificationResult
		result, err := c.RedeemChallenge(r.Context(), params.Token, params.Solutions)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp.Message = err.Error()
		} else {
			resp.Success = true
			resp.TokenData = result
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleValidateToken 检查验证令牌 Http Handle 封装
func (c *Cap) HandleValidateToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.limiter[2] != nil && !c.limiter[2].Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var params VerificationParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VerificationResult{
			Success: c.ValidateToken(r.Context(), params.Token),
		})
	}
}
