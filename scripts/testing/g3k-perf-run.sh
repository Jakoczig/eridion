#!/usr/bin/env bash
# G3K measurement set: three interleaved repetitions of every case for one or
# more labelled perf-host binaries. Usage:
#   g3k-perf-run.sh OUTDIR FIXTURE10K FIXTURE100K PLAYWRIGHT CHROMIUM label=HOST[,WEBAPP_TEST[,flags]] ...
set -u
out=$1 fx10k=$2 fx100k=$3 pw=$4 chrome=$5; shift 5
mkdir -p "$out"
cases=${G3K_CASES:-"launcher-idle dashboard-idle dashboard-hidden live-latency fsd-transitions tab-switch paired-1-same paired-2-same paired-2-different resume-reconnect"}
here=$(cd "$(dirname "$0")" && pwd)
for rep in ${G3K_REPS:-1 2 3}; do
  for spec in "$@"; do
    label=${spec%%=*}; rest=${spec#*=}
    host=$(echo "$rest" | cut -d, -f1); svc=$(echo "$rest" | cut -d, -f2 -s); flags=$(echo "$rest" | cut -d, -f3 -s)
    if [ -n "$svc" ] && [ -z "${G3K_SKIP_SERVICE:-}" ]; then
      ERIDION_PERF_JOURNAL=$fx100k "$svc" -test.run '^TestPerfNativeServiceBaseline$' -test.timeout 30m 2>&1 | grep '^PERF_RESULT ' | sed "s/^PERF_RESULT /{\"label\":\"$label\",\"rep\":$rep,\"service\":/; s/\$/}/" >> "$out/service.jsonl"
    fi
    for c in $cases; do
      node "$here/g3k-perf.cjs" --host "$host" --fixture "$fx10k" --case "$c" --rep "$rep" --out "$out/$label.jsonl" --playwright "$pw" --chromium "$chrome" $flags >> "$out/run.log" 2>&1 || echo "FAILED $label $c $rep" >> "$out/run.log"
    done
  done
done
echo DONE >> "$out/run.log"
