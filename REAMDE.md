# Backlink Engine

> The goal of this project is to enable two-way navigation on the web via backlinks. To pursue this goal, I must create an exhaustive page-level webgraph. This repository provides the tools to do so.

## Goals

### Quality and Exhaustiveness

I want the backlink list to be as correct as possible, which means that as few sources of links as possible should be missing. However, as the webgraph grows, the sources become less and less "useful," and the costs increase. The crawler should be biased toward fetching “useful” pages first and avoiding spam.  

> For the MVP, I target 5 billion nodes in the webgraph.

Exhaustiveness also implies the support for a lot of different media and formats, but this also brings a lot of complexity as each requires its own logic. For the MVI, I only target `text/html` over HTTP.

### Freshness

Freshness is important to my goal, as without it, new web pages won't be supported. Large but infrequent crawls, like the common crawl, are insufficient for my needs. This is especially true for some special types of websites, like newspapers, which add new, fresh pages continuously.  

For the MVP, I won’t implement any targeted freshness feature; instead, I will rely on sheer performance to frequently refresh the crawl.  

> My current MVP goal is for a full crawl to be done every month, meaning the data must be no more than 1 month old.

This is insufficient, but I will focus on freshness in a later stage.

### Performance and Cost

In order to achieve my MVP goals for exhaustiveness and freshness, I have set the following performance goal:  

>  1,000,000,000 requests per week  
    ~ 150,000,000 requests per day  
    ~   6,000,000 requests per hour  
    ~     100,000 requests per minute  
    ~       1,650 requests per second  

This goal would be very easy to achieve with a cluster of servers, but the cost for this project comes from my pocket, so keeping costs as low as possible is a big goal. To efficiently reach my performance goal, I will focus on scaling vertically rather than horizontally, as horizontal scaling is very expensive and will only be used as a last resort. Different infrastructure options are currently considered (and the budget that comes with them!), but no choice will be made until the prototype has advanced further.

