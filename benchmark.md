# benchamrks

> ⚠ WARNING ⚠:
    The following benchmark can vary a lot (some can vary by about ±30% req/s). They are only meant as a broad estimation so that i can keep the order of magnitude in my head.

## commit: 3e5d1184148a385a4511f0b2bf588b6ea59162c6

Setup: 
- Async collector
- Timeout 5s
- Parallelisme 8
- colly use storage from extension zolamk/colly-postgres-storage.
- I insert all result into a postgreDb using batch. 

> 
    │     Time      │   requests    │    errors     │   timeouts    │
    │           10s │          3663 │             3 │             0 │
    │           20s │          4895 │             9 │             0 │
    │           30s │          7372 │            10 │             0 │
    │           40s │          9842 │            11 │             0 │
    │           50s │         11186 │            11 │             0 │
    │          1m0s │         11469 │            11 │             0 │

191.15 req/s


# Commit c735bb23719c7c69fb1a50e806f747701cc5b2df

Setup: 
- Async collector
- Timeout 5s
- Parallelisme 8
- colly use built-in InMemoryStorage
- I insert all result into a postgreDb using batch. 
>
    │     Time      │   requests    │    errors     │   timeouts    │
    │           10s │          3769 │             3 │             1 │
    │           20s │          7852 │            17 │             2 │
    │           30s │          9017 │            56 │             2 │
    │           40s │         10417 │            56 │             2 │
    │           50s │         11384 │            56 │             2 │
    │          1m0s │         12316 │            56 │             2 │

205.3 req/s
