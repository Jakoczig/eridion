#!/usr/bin/env python3
"""Summarise G3K performance runs (scripts/testing/g3k-perf-run.sh output).

usage: g3k-perf-report.py DIR [DIR ...]
Prints per case and label: mean, min and max over repetitions of the key
metrics, as Markdown tables. Raw JSON lines stay the source of truth.
"""
import json
import os
import statistics
import sys


def load(dirs):
    rows, service = [], []
    for d in dirs:
        for name in sorted(os.listdir(d)):
            path = os.path.join(d, name)
            if name == 'service.jsonl':
                service += [json.loads(l) for l in open(path) if l.strip()]
            elif name.endswith('.jsonl'):
                label = name[:-6]
                for l in open(path):
                    if l.strip():
                        r = json.loads(l)
                        r['label'] = label + ('-lowlatency' if r.get('lowLatency') else '')
                        rows.append(r)
    return rows, service


def metrics(r):
    c = r.get('host', {}).get('counters', {})
    t = r.get('host', {}).get('timings', {})
    out = {}
    w = r.get('window')
    if w:
        out['host CPU s'] = w['hostCPUSeconds']
        out['browser CPU s'] = w['browserCPUSeconds']
        out['host PSS MB (mean)'] = w['hostPSSMB']['mean']
        out['browser PSS MB (mean)'] = w['browserPSSMB']['mean']
    if 'cpu' in r:
        out['host CPU s'] = r['cpu']['host']
        out['browser CPU s'] = r['cpu']['browser']
    sock = r.get('socket')
    if isinstance(sock, dict):
        out['socket frames'] = sock['frames']
        out['socket KB'] = round(sock['bytes'] / 1024, 1)
    elif isinstance(sock, list):
        out['socket frames (all displays)'] = sum(s['frames'] for s in sock)
        out['socket KB (all displays)'] = round(sum(s['bytes'] for s in sock) / 1024, 1)
        for i, s in enumerate(sock):
            out[f'frames display {i}'] = s['frames']
    if 'latencyMs' in r:
        out['latency p50 ms'] = r['latencyMs']['p50']
        out['latency p95 ms'] = r['latencyMs']['p95']
    if 'tabSwitchMs' in r:
        for route, v in r['tabSwitchMs'].items():
            out[f'tab {route} p50 ms'] = v['p50']
    if 'hiddenSocket' in r:
        out['hidden frames/60s'] = r['hiddenSocket']['frames']
        out['hidden KB/60s'] = round(r['hiddenSocket']['bytes'] / 1024, 1)
        out['resume to fresh ms'] = r['resumeToFreshMs']
        out['reconnect to fresh ms'] = r['reconnectToFreshMs']
    if 'pss' in r:
        out['host PSS MB'] = r['pss']['hostMB']
        out['browser PSS MB'] = r['pss']['browserMB']
    out['projection builds'] = c.get('projection.build', 0)
    out['domain builds'] = sum(v for k, v in c.items() if k.startswith('domain.build.'))
    out['context builds'] = c.get('context.build', 0)
    out['shared encodings'] = sum(v for k, v in c.items() if k.startswith('ws.shared.'))
    hold = t.get('game.mu.hold')
    if hold:
        out['lock hold total ms'] = round(hold['sumMs'], 1)
        out['lock hold max ms'] = round(hold['maxMs'], 1)
    return out


def fmt(values):
    if not values:
        return ''
    if len(values) == 1:
        return f'{values[0]:g}'
    return f'{statistics.mean(values):.4g} ({min(values):g}–{max(values):g})'


def main():
    rows, service = load(sys.argv[1:])
    cases = sorted({r['case'] for r in rows})
    labels = sorted({r['label'] for r in rows})
    for case in cases:
        per = {lab: [metrics(r) for r in rows if r['case'] == case and r['label'] == lab] for lab in labels}
        per = {k: v for k, v in per.items() if v}
        keys = []
        for ms in per.values():
            for m in ms:
                for k in m:
                    if k not in keys:
                        keys.append(k)
        print(f'\n### {case}\n')
        print('| metric | ' + ' | '.join(f'{lab} (n={len(per[lab])})' for lab in per) + ' |')
        print('| --- |' + ' --- |' * len(per))
        for k in keys:
            print(f'| {k} | ' + ' | '.join(fmt([m[k] for m in per[lab] if k in m]) for lab in per) + ' |')
    if service:
        print('\n### headless service (100k events)\n')
        labs = sorted({s['label'] for s in service})
        keys = ['coldImportSeconds', 'coldImportCPUSeconds', 'coldPeakWorkingSetMB', 'incrementalRestartSeconds', 'incrementalRestartCPUSeconds', 'liveAppendP50ms', 'liveAppendP95ms', 'liveAppendMaxms', 'runPeakWorkingSetMB']
        print('| metric | ' + ' | '.join(f'{lab} (n={sum(1 for s in service if s["label"] == lab)})' for lab in labs) + ' |')
        print('| --- |' + ' --- |' * len(labs))
        for k in keys:
            print(f'| {k} | ' + ' | '.join(fmt([s['service'][k] for s in service if s['label'] == lab]) for lab in labs) + ' |')
        for phase in ('coldPerfstats',):
            for name in ('checkpoint.commit', 'checkpoint.serialize', 'game.mu.hold', 'game.mu.wait'):
                print(f'| cold {name} total ms | ' + ' | '.join(fmt([round(s['service'][phase]['timings'][name]['sumMs'], 1) for s in service if s['label'] == lab and name in s['service'][phase]['timings']]) for lab in labs) + ' |')
                print(f'| cold {name} max ms | ' + ' | '.join(fmt([round(s['service'][phase]['timings'][name]['maxMs'], 2) for s in service if s['label'] == lab and name in s['service'][phase]['timings']]) for lab in labs) + ' |')


if __name__ == '__main__':
    main()
