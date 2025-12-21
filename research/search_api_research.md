# Research Report: Building vs. Buying a Search API

## Executive Summary
Building a full-scale internet search engine from scratch is a monumental task requiring significant infrastructure, expertise, and capital ($100M+ for a Google-scale prototype). However, creating a **niche search engine** or a **meta-search engine** is a feasible project for a solo developer or small team. Alternatively, using an AI-native search API like **Exa (formerly Metaphor)** provides immediate access to high-quality, semantic functionality optimized for LLMs.

This document outlines the technical requirements for building a search API, explores the "Meta-Search" alternative, and details the capabilities of the Exa API.

---

## Option 1: Building a Simple Search Engine (Golang Architecture)

To build a search engine that actually crawls and indexes the web (even a small subset), you need three core components. Golang is an excellent choice for this due to its concurrency model.

### 1. The Crawler ("Spider")
*   **Function:** Traverses the web, fetching HTML content.
*   **Golang Tools:** `Colly` (flexible framework), `chromedp` (if JS rendering is needed).
*   **Challenges:** 
    *   **Politeness:** Respecting `robots.txt` and rate limits.
    *   **Scale:** Handling millions of URLs requires distributed crawling (using extensive Goroutines and distributed message queues like Kafka or NATS).
    *   **Storage:** Storing raw HTML (WARC files) requires massive disk space (Common Crawl is PB scale).

### 2. The Indexer
*   **Function:** Parses content and creates a searchable data structure (Inverted Index).
*   **Mechanism:** Maps terms (keywords) to document IDs. Requires tokenization, stemming (snowball), and stop-word removal.
*   **Golang Tools:** `Bleve` (native Go text indexing library).
*   **Infrastructure:** For any serious scale, you would likely offload this to **Elasticsearch**, **Meilisearch**, or **Solr** rather than writing a raw index from scratch.
*   **Challenge:** Keeping the index fresh. The web changes constantly.

### 3. The Search & Ranking Engine
*   **Function:** Takes a user query, looks up terms in the index, and ranks results.
*   **Algorithms:** 
    *   **TF-IDF / BM25:** Standard relevance scoring.
    *   **PageRank:** Graph-based scoring based on backlinks (extremely compute-intensive).
*   **Challenge:** "Google-level" quality is incredibly hard. Dealing with spam, SEO optimizers, and understanding user intent (semantic search) requires advanced ML/AI models, not just keyword matching.

### **Verdict:**
*   **Feasibility:** Low for a general internet search. High for a **niche** search (e.g., "golang blog posts only").
*   **Cost:** High (Storage + Compute).

---

## Option 2: The Meta-Search Engine Approach

Instead of crawling the web, a meta-search engine queries *other* search engines (Google, Bing, DuckDuckGo) and aggregates the results.

### Architecture
1.  **Proxy Layer:** Your Go API receives a query.
2.  **Dispatcher:** Sends requests to multiple upstream providers (e.g., Bing API, Programmable Search Engine, Serper.dev).
3.  **Aggregator:** Normalizes the different result formats into a single JSON response.
4.  **Re-ranker:** (Optional) You can re-rank results based on your own logic or filter them.

### Pros & Cons
*   **Pros:** strict control over the API shape, no storage required, high result quality immediately.
*   **Cons:** You are just wrapping other APIs. If you scrape them (without API keys), you will be blocked instantly by captchas. If you use their APIs, it costs money (e.g., Bing Web Search API is ~$25/1k calls).

---

## Option 3: Using Exa (formerly Metaphor)

Exa is distinct because it is an **Embedding-based Search Engine**. It doesn't just match keywords; it understands meaning. It is built specifically for AI agents and LLMs.

### Key Features
*   **Semantic Search:** Finds results that *mean* the same thing, even if words differ.
*   **Clean Content:** Returns widely clean HTML/Text, saving you from building a separate scraper/parser.
*   **Neural Search:** "Search by example" (provide a URL, get similar URLs).
*   **Filterable:** Robust filters for domains, dates, and content types.

### Why use Exa?
If the goal is to feed an LLM or Agent, Exa is superior to Google. Google gives you 10 blue links optimized for ad-clicks. Exa gives you structured content optimized for machine reading.

---

## Recommendation

1.  **If you need a generic "Google" for your app:**
    *   Do **NOT** build from scratch.
    *   Use a **Meta-Search API** wrapper like **Serper.dev** (cheaper Google wrapper) or **Bing Search API**.

2.  **If you are feeding an AI Agent (Context Retrieval):**
    *   Use **Exa.ai**. It solves the retrieval + scraping step in one request.

3.  **If you need to search a specific list of domains (e.g., "Search these 50 documentation sites"):**
    *   Build a **Crawl-based Engine** using **Colly** + **Meilisearch**. This is cost-effective and strictly scoped.

4.  **If you want to create a privacy-focused wrapper:**
    *   Build a **Meta-Search Engine** in Go that aggregates results from specific providers, strips trackers, and relays the clean JSON.
