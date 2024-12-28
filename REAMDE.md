# Backlink engine

> The goal of this project is to enable the two way navition on the web via backlinks. To pursue this goal we must create an exaustive page-level webgraph. This repos provide the tools to do so.

## Goals

### Quality and Exaustiveness 

We want our backlink list to be as correct as possible, wich mean that as little source of links as possible should be missing. But, as the webgraph grow, the sources become less and less "useful" and the costs increases. The crawler should be biased towards fetching “useful” pages first and avoid spam. 

> For the MVP we target 5 billions nodes in ours webgraph.

Exaustiveness also implies the support for a lot of different media and format but this also bring a lot of complexity as each require it's own logic. For the MVI we only target `text/html` over HTTP. 

### Freshness

Freshness is important to ours goal as without it new web pages won't be supported. Large but infrequent crawl like the common crawl is insufficient for ours need. This is especialy true for some special type of website, like newspaper, wich add new fresh page continuously. 

For the MVP we wont implement any targeted freshness feature, instead we will rely on shear performance to frequently refresh the crawl. 

> Our current MVP goal is for a full crawl to be done every mounth, meaning the data must be no more than 1 month old.

This is insufficient but we will focus on freshness in a later stage.

### Performance and Cost

In order to achieve our MVP goal for exaustiveness and freshness we have set the following performance goal:  

>  1 000 000 000 request per week  
    ~ 150 000 000  request per day  
    ~   6 000 000 request per hour  
    ~     100 000 request per minute  
    ~       1 650 request per second  

This goal would be very easy to achieve with cluster of server but the cost for this project come from my pocket so keeping cost as low as possible is a big goal. To efficiently reach our performance goal we will focus on scalling vertically rather than horizontally as horizontal scalling is very expensive and as such will only be used a last resort. Different infrastructure options are currently concidered (and the budget that come with it!) but no choice will be made until the prototype has advanced further. 

