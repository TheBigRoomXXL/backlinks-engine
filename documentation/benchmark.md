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


# Commit 6daa480c65cac55f545d7b1a84ed46a9a9b46a5c (memory issues)

    ❯ go run . crawl http://localhost/seeds
    2024/12/30 09:09:58 Starting the pprof server of port  8081
    ┌───────────────┬───────────────┬───────────────┬───────────────┬───────────────┐
    │     Time      │   Processed   │   Alloc MB    │ TotalAloc MB  │   Goroutine   │
    ├───────────────┼───────────────┼───────────────┼───────────────┼───────────────┤
    │            1s │          3268 │ 660           │ 929           │ 194669        │
    │            2s │          4794 │ 1315          │ 2070          │ 605107        │
    │            3s │          7635 │ 2010          │ 2919          │ 831620        │
    │            4s │         10962 │ 2886          │ 4247          │ 1148624       │
    │            5s │         12964 │ 3494          │ 5048          │ 1376460       │
    │            6s │         15114 │ 4067          │ 6426          │ 1773000       │
    │            7s │         17434 │ 5071          │ 7431          │ 2083273       │
    │            8s │         20193 │ 6188          │ 8684          │ 2520683       │
    │            9s │         22490 │ 6185          │ 9678          │ 2872432       │
    │           10s │         26080 │ 6853          │ 10932         │ 3298745       │
    │           11s │         30036 │ 8180          │ 12259         │ 3669659       │
    │           12s │         32090 │ 8986          │ 13065         │ 3889588       │
    │           13s │         33759 │ 9572          │ 13652         │ 4059506       │
    │           15s │         34913 │ 10738         │ 14817         │ 4396747       │
    │           15s │         36564 │ 11033         │ 15129         │ 4492751       │
    │           16s │         38462 │ 11027         │ 16358         │ 5020998       │
    │           17s │         41900 │ 11024         │ 17526         │ 5332976       │
    │           18s │         44611 │ 11153         │ 18585         │ 5583177       │
    │           19s │         46930 │ 12289         │ 19846         │ 5910293       │
    signal: killed
