# Examples — AHP price integrity

## Fail: US price converted

```text
set-attribute samson_sr850 value 300 --unit BRL --source "Amazon.com $59≈R$"
```

**Why fail:** foreign listing + FX guess. User complaint: “preços de fontes americanas convertidas”.

**Fix:** King Musical boleto R$368.99 (or another BR specialty quote) with that source named.

## Fail: marketplace 3P outlier

```text
set-attribute shure_srh240a value 359 --unit BRL --source "KaBuM marketplace 3P"
```

Meanwhile Magalu/Buscapé ~R$505.

**Why fail:** 3P too-cheap listing; also **out of R$200–400** on reputable quotes.

**Fix:** drop from shortlist or replace (e.g. AKG K414P R$315 King Musical).

## Fail: pairwise contradicts attributes

Attributes: A=289, B=369 (prefer lower).  
Pair: `alt:value B A 3` (claims B better value).

**Why fail:** direction inverted vs objective prices.

**Fix:** `ahp rate --criterion value --prefer lower` or set `A B 2` (A cheaper → A preferred).

## Pass: local Pix/boleto band

| Alt | Value | Source |
|-----|-------|--------|
| superlux_hd681 | 289 | Cheda's SP Pix |
| akg_k414p | 315 | King Musical boleto |
| akg_k52 | 338 | Serenata BH |
| samson_sr850 | 369 | King Musical boleto |

All in R$200–400, sources local, then `rate --prefer lower` / consistent `alt:value`.
