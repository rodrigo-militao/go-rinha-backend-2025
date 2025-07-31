//go:build !release

package pprof

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		log.Println("pprof ativo em :6060")
		err := http.ListenAndServe("0.0.0.0:6060", nil)
		if err != nil {
			log.Fatalf("Erro ao iniciar pprof: %v", err)
		}
	}()
}
