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
- seeds: https://www.bbc.com/, https://www.theguardian.com/europe/, https://www.liberation.fr/

> 
    ┌───────────────┬───────────────┬───────────────┬───────────────┐
    │     Time      │   requests    │    errors     │   timeouts    │
    ├───────────────┼───────────────┼───────────────┼───────────────┤
    │           10s │          3663 │             3 │             0 │
    │           20s │          4895 │             9 │             0 │
    │           30s │          7372 │            10 │             0 │
    │           40s │          9842 │            11 │             0 │
    │           50s │         11186 │            11 │             0 │
    │          1m0s │         11469 │            11 │             0 │
    └───────────────┴───────────────┴───────────────┴───────────────┘

191.15 req/s


# Commit c735bb23719c7c69fb1a50e806f747701cc5b2df

Setup: 
- Async collector
- Timeout 5s
- Parallelisme 8
- colly use built-in InMemoryStorage
- I insert all result into a postgreDb using batch. 
- seeds: https://www.bbc.com/, https://www.theguardian.com/europe/, https://www.liberation.fr/
>
    ┌───────────────┬───────────────┬───────────────┬───────────────┐
    │     Time      │   requests    │    errors     │   timeouts    │
    ├───────────────┼───────────────┼───────────────┼───────────────┤
    │           10s │          3769 │             3 │             1 │
    │           20s │          7852 │            17 │             2 │
    │           30s │          9017 │            56 │             2 │
    │           40s │         10417 │            56 │             2 │
    │           50s │         11384 │            56 │             2 │
    │          1m0s │         12316 │            56 │             2 │
    └───────────────┴───────────────┴───────────────┴───────────────┘

205.3 req/s


# Commit 546685415d49a3e83d5c26093a40459d5c9d2199

Setup: 
- Async collector
- Timeout 5s
- Parallelisme 8
- colly use built-in InMemoryStorage
- No insert at all, we don't keep the data, we only scrap
- seeds: https://www.bbc.com/, https://www.theguardian.com/europe/, https://www.liberation.fr/

> 
    ┌───────────────┬───────────────┬───────────────┬───────────────┐
    │     Time      │   requests    │    errors     │   timeouts    │
    ├───────────────┼───────────────┼───────────────┼───────────────┤
    │           10s │          3825 │             3 │             0 │
    │           20s │         10592 │             7 │             0 │
    │           30s │         11210 │            10 │             0 │
    │           40s │         11577 │            10 │             0 │
    │           50s │         12391 │            10 │             0 │
    │          1m0s │         13168 │            10 │             0 │
    └───────────────┴───────────────┴───────────────┴───────────────┘

219 req/s


# Commit 20261a3438e4b65376268cb19b3a47ce14ba11b4


Setup: 
- Async collector
- Timeout 5s
- Parallelisme 8
- colly use built-in InMemoryStorage
- Data Inserted into Neo4J
- seeds: https://www.bbc.com/, https://www.theguardian.com/europe/, https://www.liberation.fr/


> 
    ┌───────────────┬───────────────┬───────────────┬───────────────┐
    │     Time      │   requests    │    errors     │   timeouts    │
    ├───────────────┼───────────────┼───────────────┼───────────────┤
    │           10s │          5337 │             2 │             0 │
    │           20s │          7950 │             3 │             0 │
    │           30s │         16089 │             5 │             0 │
    │           40s │         20330 │            17 │             1 │
    │           50s │         25868 │            25 │             1 │
    │          1m0s │         32825 │            44 │             2 │
    │         1m10s │         35347 │           237 │           189 │
    │         1m20s │         38983 │           556 │           505 │
    │         1m30s │         42696 │           900 │           840 │
    │         1m40s │         43407 │          1184 │          1123 │
    └───────────────┴───────────────┴───────────────┴───────────────┘

434 req/s

The timeout come from neo4j transactions
