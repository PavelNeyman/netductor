# VPS tests (English)

**RU:** [ru/VPS-TESTS.md](ru/VPS-TESTS.md)

Unified runner wraps common community probes. Scripts run in a **temp dir** and are **deleted on exit** (trap). Summary table is printed to the terminal only.

## Run

```bash
sudo netductor probe                 # live probes
# curated third-party scripts: see docs/VPS-TESTS history / operator scripts
sudo netductor probe        # curated subset
sudo netductor probe            # long
sudo netductor probe yabs sysbench
sudo bash install.sh --tests        # same menu from installer
```

| Id | Source |
|----|--------|
| ipregion | ipregion.vrnt.xyz |
| censor_geoblock / censor_dpi | vernette/censorcheck |
| iperf_ru | itdoginfo/russian-iperf3-servers |
| yabs | yabs.sh |
| ip_check_place | IP.Check.Place |
| bench | bench.sh |
| ipquality | Check.Place -EI |
| sysbench | local package |

**Note:** Third-party installers may leave system packages (e.g. `sysbench`). We only wipe our temp workspace, not apt packages they installed.

Trust model: you are piping remote scripts; use on throwaway/test VPS when unsure.
