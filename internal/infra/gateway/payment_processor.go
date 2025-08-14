package gateway

import (
	"rinha-golang/internal/domain"

	"github.com/valyala/fasthttp"
)

// func PostPayment(client *http.Client, p *domain.PaymentRequest, url string) bool {
// 	data := map[string]any{
// 		"correlationId": p.CorrelationId,
// 		"amount":        p.Amount,
// 		"requestedAt":   p.RequestedAt,
// 	}

// 	// TODO: trocar para easyjson
// 	payload, _ := json.Marshal(data)

// 	res, _ := client.Post(
// 		url,
// 		"application/json",
// 		bytes.NewBuffer(payload),
// 	)

// 	defer res.Body.Close()

// 	_, _ = io.Copy(io.Discard, res.Body)

// 	if res.StatusCode != 200 && res.StatusCode != 422 {
// 		return false
// 	}

// 	return true
// }

func PostPayment(client *fasthttp.HostClient, payment *domain.PaymentRequest) bool {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	uri := "http://" + client.Addr + "/payments"

	req.SetRequestURI(uri)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")

	body, _ := payment.MarshalJSON()
	req.SetBodyRaw(body)

	if err := client.Do(req, resp); err != nil {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return false
	}

	ok := resp.StatusCode() >= 200 && resp.StatusCode() < 300
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)

	return ok
}
