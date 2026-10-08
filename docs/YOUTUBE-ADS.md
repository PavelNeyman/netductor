**EN** · optional

# YouTube ads vs netductor DNS (blocky)

## Reality (2026)

**In-stream YouTube ads cannot be blocked reliably by DNS alone.**  
Ads and video share `*.googlevideo.com` / same CDN. Blocking those hostnames breaks playback. NextDNS and HaGezi maintainers state the same limit. Server-side ad insertion (SSAI) makes DNS even less effective.

What **blocky + HaGezi/StevenBlack/AdGuard** *does* on VPN:

- Trackers, doubleclick, many banners on *other* sites  
- Some YouTube *chrome* trackers / non-video ad domains  
- Network-wide when client DNS goes through secondary/blocky  

What it **does not** do:

- Pre-roll / mid-roll video ads in official YouTube app / TV  
- SSAI-stitched ads  

## Options that fit “connect VPN → works”

| Option | Auto on VPN | Reliability for YT video ads | Fit |
|--|--|--|
| **Keep blocky lists** (status quo) | Yes | Partial / often weak for YT | Keep — still useful overall |
| Extra list [kboghdady/youTube_ads_4_pi-hole](https://github.com/kboghdady/youTube_ads_4_pi-hole) | Yes | Partial, updates daily, can false-positive | Optional experiment on secondary |
| **YouTube Premium** | N/A (account) | Full official | Best for family phones |
| Browser uBlock / ReVanced / NewPipe | Per device | High | Not “VPN only” |
| MITM HTTPS filter | Could be VPN-side | Fragile, heavy, privacy risk | **Reject** for netductor |

## Recommendation for netductor

1. Do **not** promise clean YouTube via DNS.  
2. Ensure VPN clients use **blocky** (not system DoH bypass).  
3. Optional: add curated YT list as a **toggle** in DNS config, default off, with doctor note.  
4. Document Premium / client-side for real ad-free video.

No MITM TLS interception on primary.
