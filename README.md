# Rinha de Backend 2025 - Implementação em Go

Esta é a minha submissão para a 3ª edição da Rinha de Backend, desenvolvida em Go. O objetivo foi construir uma API de pagamentos resiliente e de alta performance, explorando padrões de arquitetura de software e otimizações.

## ✨ Tecnologias Utilizadas

- **Linguagem:** Go 1.22+
- **Cache / Banco de Dados de Estado:** Redis 7
- **Fila de Mensagens:** Redis 7 (usando Listas)
- **Load Balancer:** Nginx
- **Ferramentas:** Docker & Docker Compose, k6

## 🏛️ Arquitetura

A arquitetura escolhida foi um sistema **assíncrono** projetado para alta vazão e baixa latência na API, com o Redis no centro de todas as operações de estado e mensageria.

- **Nginx:** Atua como Reverse Proxy e Load Balancer (`round_robin`), distribuindo as requisições para duas instâncias da aplicação Go.
- **Aplicação Go:** Duas instâncias independentes. A API (`POST /payments`) é extremamente leve: sua única responsabilidade é publicar o trabalho em uma fila no Redis.
- **Fila de Trabalho (Redis Lists):** Uma lista (`payments_queue`) serve como uma fila de trabalho central. O padrão `LPUSH`/`BLPOP` garante que cada trabalho seja pego por apenas um worker.
- **Workers:** Um pool de `goroutines` em cada instância da aplicação consome os trabalhos da fila em paralelo. Cada worker é responsável pela lógica de negócio: idempotência (via `SetNX`), circuit breaking e a comunicação com os processadores de pagamento externos.
- **Armazenamento de Estado:** O estado final dos pagamentos processados é salvo em um Redis Hash para leituras rápidas e escritas idempotentes, com summaries pré-agregados por segundo para consultas eficientes.

```mermaid
graph TD
    subgraph "Cliente (k6)"
        A[Requisição HTTP]
    end

    subgraph "Infraestrutura"
        A --> B(Nginx Load Balancer);
        B -- least_conn --> C1(App Go 1);
        B -- least_conn --> C2(App Go 2);
    end

    subgraph "Processamento Assíncrono com Redis"
        C1 -- LPUSH --> D -> Redis (Fila de Trabalho);
        C2 -- LPUSH --> D;
        D -- BLPOP --> E1(Worker Pool - App 1);
        D -- BLPOP --> E2(Worker Pool - App 2);
    end

    subgraph "Dependências"
        E1 -- HSet/SetNX --> H[Redis (Estado/Idempotência)];
        E2 -- HSet/SetNX --> H;
        E1 -- Processa Pagamento --> G(Processadores Externos);
        E2 -- Processa Pagamento --> G;
    end
```

## 🚀 Como Executar Localmente

1.  Clone este repositório.
2.  Certifique-se de ter o Docker e o Docker Compose instalados.
3.  Execute o ambiente:
    ```sh
    # O arquivo docker-compose.yml padrão já está configurado para o Redis.
    docker-compose up --build
    ```
4.  Em outro terminal, execute o teste de carga com o k6:
    ```sh
    k6 run rinha-test/rinha.js
    ```

## 👤 Autor

**[Rodrigo Militão]**
- x: [@RodrigoMilitao8](https://x.com/RodrigoMilitao8)
- LinkedIn: [rodrigo-militao](https://linkedin.com/in/rodrigo-militao)