package gateway

import (
	"rinha-golang/internal/domain"

	"github.com/valyala/fasthttp"
)

// func PostPayment(client *http.Client, p *domain.PaymentRequest, url string) bool {
// 	body, _ := p.MarshalJSON()

// 	res, err := client.Post(
// 		url,
// 		"application/json",
// 		bytes.NewBuffer(body),
// 	)

// 	if err != nil {
// 		return false
// 	}

// 	defer res.Body.Close()

// 	_, _ = io.Copy(io.Discard, res.Body)

// 	return res.StatusCode >= 200 && res.StatusCode < 300
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

// func PostPayment(client *fasthttp.Client, payment *domain.PaymentRequest, url string) bool {
// 	req := fasthttp.AcquireRequest()
// 	resp := fasthttp.AcquireResponse()

// 	req.SetRequestURI(url)
// 	req.Header.SetMethod(fasthttp.MethodPost)
// 	req.Header.SetContentType("application/json")

// 	body, _ := payment.MarshalJSON()
// 	req.SetBodyRaw(body)

// 	if err := client.Do(req, resp); err != nil {
// 		fasthttp.ReleaseRequest(req)
// 		fasthttp.ReleaseResponse(resp)
// 		return false
// 	}

// 	ok := resp.StatusCode() >= 200 && resp.StatusCode() < 300
// 	fasthttp.ReleaseRequest(req)
// 	fasthttp.ReleaseResponse(resp)

// 	return ok
// }
