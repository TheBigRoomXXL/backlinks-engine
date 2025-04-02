# Backlink Engine 
## This project is a work in progress

> The goal of this project is to enable two-way navigation on the web via backlinks. To pursue this goal, I must create an exhaustive page-level webgraph. This repository provides the tools to do so.

## Goals

### Quality and Exhaustiveness

I want the backlink list to be as accurate as possible, which means that as few sources of links as possible should be missing. However, as the webgraph grows, the sources become less and less "useful," and the costs increase. The crawler should be biased toward fetching “useful” pages first and avoiding spam.

> For the MVP, I target 5 billion nodes in the webgraph.

Exhaustiveness also implies support for a lot of different media and formats, but this brings a lot of complexity, as each format requires its own logic. For the MVP, I only target `text/html` over HTTP.

### Freshness

Freshness is important to my goal because, without it, new web pages won’t be supported. Large but infrequent crawls, like the common crawl, are insufficient for my needs. This is especially true for some special types of websites, like newspapers, which continuously add new, fresh pages.

For the MVP, I won’t implement any targeted freshness features; instead, I will rely on sheer performance to frequently refresh the crawl.

> My current MVP goal is to complete a full crawl every month, meaning the data must be no more than 1 month old.

This is insufficient, but I will focus on freshness in a later stage.

### Performance and Cost

To achieve my MVP goals for exhaustiveness and freshness, I have set the following performance goals:

> 1,000,000,000 requests per week  
> ~ 150,000,000 requests per day  
> ~ 6,000,000 requests per hour  
> ~ 100,000 requests per minute  
> ~ 1,650 requests per second

This goal would be very easy to achieve with a cluster of servers, but the cost for this project comes from my pocket, so keeping costs as low as possible is a big priority. To efficiently reach my performance goal, I will focus on scaling vertically rather than horizontally, as horizontal scaling is very expensive and will only be used as a last resort. Different infrastructure options are currently being considered (and the budgets that come with them!), but no final decision will be made until the prototype has advanced further.

On the database side, this translates to the following approximate requirements:

#### Links Table
- On the order of 10^11 rows
- Requests to individual targets must be completed in under ~200ms
- Support ~100k row inserts per second
- Ability to delete obsolete records (eventual consistency is acceptable)

#### Pages Table
- On the order of 10^9 rows
- Used as a queue
- Store, at minimum, structured URLs and visit timestamps

#### Domain Table
- On the order of 10^7 rows
- Store `robots.txt`
- Store aggragated statistics

I am going to start with a simple implementation with postgres and then migrate as much as possible data to Clickhouse. I am uncertain that all data can be migrated as it lack some transactional features. Clickhouse compression is going to be essential to keep the storage costs down.
