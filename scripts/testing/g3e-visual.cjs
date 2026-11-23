const fs = require('node:fs/promises')
const path = require('node:path')
const { fixture, browser, waitFor, delay, pageTextReady } = require('./g2d-visual.cjs')
const root = path.resolve(__dirname, '../..')
async function main () {
  const output = path.join(root, 'temp/g3e-visual'); await fs.mkdir(output, { recursive: true })
  const report = { scope: 'Native loopback UI; synthetic journal and explicitly injected spatial response, not gameplay or publisher validation', errors: [], external: [], layouts: [] }
  const b = await browser(output); const c = b.client; let host; let origin = ''
  const evaluate = async expression => { const r = await c.call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }); if (r.exceptionDetails) throw Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text); return r.result.value }
  const click = text => evaluate(`Array.from(document.querySelectorAll('.go-galaxy-sidebar button, .go-fsd>.go-page__heading button')).find(b=>b.textContent===${JSON.stringify(text)})?.click()`)
  // Task 3 / F03: the actual cascade must give every map, including the FSD
  // overlay, the canonical three-panel grid: square X/Z, X/Y as wide as X/Z,
  // Y/Z as tall as X/Z, equal thin panels, all visible in the viewport.
  const geometry = scope => evaluate(`(()=>{const root=document.querySelector(${JSON.stringify(scope)});if(!root)return null;const p=root.querySelector('.go-galaxy-projections');const box=e=>{if(!e)return null;const r=e.getBoundingClientRect();return {left:r.left,top:r.top,right:r.right,bottom:r.bottom,width:r.width,height:r.height}};const svg=n=>box(root.querySelector('.go-galaxy-'+n+'>svg'));return {display:p&&getComputedStyle(p).display,xz:svg('xz'),xy:svg('xy'),yz:svg('yz'),vw:innerWidth,vh:innerHeight,mode:root.dataset.mode}})()`)
  const checkGeometry = (g, label) => {
    const bad = []
    if (!g) bad.push('map missing')
    else {
      if (g.display !== 'grid') bad.push('projections display ' + g.display)
      for (const k of ['xz', 'xy', 'yz']) { const r = g[k]; if (!r || r.width < 8 || r.height < 8) bad.push(k + ' not visible'); else if (r.left < -1 || r.top < -1 || r.right > g.vw + 1 || r.bottom > g.vh + 1) bad.push(k + ' outside viewport') }
      if (!bad.length) {
        if (Math.abs(g.xz.width - g.xz.height) > 1.5) bad.push('X/Z not square')
        if (Math.abs(g.xy.width - g.xz.width) > 1.5) bad.push('X/Y width differs from X/Z')
        if (Math.abs(g.yz.height - g.xz.height) > 1.5) bad.push('Y/Z height differs from X/Z')
        if (Math.abs(g.xy.height - g.yz.width) > 1.5) bad.push('thin panels differ')
      }
    }
    if (bad.length) throw Error('Panel geometry ' + label + ': ' + bad.join(', ') + ' ' + JSON.stringify(g))
    return g
  }
  const sameRects = (a, b) => ['xz', 'xy', 'yz'].every(k => ['left', 'top', 'width', 'height'].every(f => Math.abs(a[k][f] - b[k][f]) <= 1))
  const fsdScope = '[aria-label="Automatic FSD display"] .go-g3d'
  const navMode = async () => (await rpc('getGoNavigationContext'))?.navigation?.mode || ''
  const shot = async name => { const r = await c.call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false }); await fs.writeFile(path.join(output, name + '.png'), Buffer.from(r.data, 'base64')) }
  const rpc = (name, message) => evaluate(`new Promise((resolve,reject)=>{const w=new WebSocket(location.origin.replace('http:','ws:')+'/?scope=launcher');w.onopen=()=>w.send(JSON.stringify({requestId:'g3e',name:${JSON.stringify(name)},message:${JSON.stringify(message)}}));w.onmessage=e=>{const r=JSON.parse(e.data);if(r.requestId==='g3e'){w.close();r.error?reject(Error(r.error)):resolve(r.message)}};w.onerror=reject})`)
  c.on(e => {
    if (e.method === 'Runtime.exceptionThrown') report.errors.push(e.params.exceptionDetails.exception?.description)
    if (e.method === 'Fetch.requestPaused') { const p = e.params; const ok = p.request.url.startsWith(origin + '/') || p.request.url.startsWith('data:'); if (!ok) report.external.push(p.request.url); c.call(ok ? 'Fetch.continueRequest' : 'Fetch.failRequest', ok ? { requestId: p.requestId } : { requestId: p.requestId, errorReason: 'BlockedByClient' }).catch(() => {}) }
  })
  await c.call('Network.enable'); await c.call('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  // Only the browser spatial-view response is synthetic. All owner settings,
  // navigation, FSD episodes, permissions and lifecycle use the real Go fixture.
  await c.call('Page.addScriptToEvaluateOnNewDocument', { source: 'window.g3eRequests=[];const send=WebSocket.prototype.send;WebSocket.prototype.send=function(raw){let q;try{q=JSON.parse(raw)}catch{};if(q?.name===\'getGoSpatialView\'){window.g3eRequests.push(q);if(window.g3eLocal){queueMicrotask(()=>this.onmessage?.({data:JSON.stringify({requestId:q.requestId,message:{complete:true,pending:false,points:[[3,4,5],[8,4,5]],notice:\'SYNTHETIC UI FIXTURE\'}})}));return}}return send.call(this,raw)}' })
  try {
    host = await fixture('G2E', '', path.join(root, 'tests/testdata/g2g')); origin = host.url
    await c.call('Page.navigate', { url: origin + '/nav/galaxy' }); await waitFor(() => evaluate(pageTextReady('Galaxy Map', true)))
    const searchButtons = await evaluate('Array.from(document.querySelectorAll(".go-galaxy-sidebar form button")).map(b=>b.textContent)')
    if (searchButtons.join('|') !== 'Search') throw Error('Search is not a single action')
    const setQuery = value => evaluate(`(()=>{const e=document.querySelector('[aria-label="Galaxy system search"]');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,${JSON.stringify(value)});e.dispatchEvent(new Event('input',{bubbles:true}))})()`)
    await setQuery('   ')
    if (!await evaluate('document.querySelector(".go-galaxy-sidebar form button").disabled')) throw Error('Whitespace search enabled')
    await setQuery('Fixture Alpha')
    await waitFor(() => evaluate('!document.querySelector(".go-galaxy-sidebar form button").disabled'))
    await evaluate('document.querySelector(\'[aria-label="Galaxy system search"]\').focus()')
    await c.call('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Enter', code: 'Enter', text: '\r', unmodifiedText: '\r', windowsVirtualKeyCode: 13, nativeVirtualKeyCode: 13 })
    await c.call('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13 })
    await waitFor(() => evaluate('document.querySelectorAll("[data-marker=searched]").length===3'))
    report.enterSearch = true
    const config = { charge: false, jump: false, restore: true, owner: true, allViewers: false, devices: [], closeMode: 'tray', galaxyMap: true, localAtlas: true, mapDuringFSD: true }
    await rpc('setGoCompanion', config)
    for (const [name, width, height] of [['desktop', 1920, 1080], ['wide', 1600, 900], ['laptop', 1366, 768], ['touch', 820, 1180], ['compact', 680, 580]]) {
      await c.call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false })
      await evaluate('document.fonts.ready.then(()=>true)'); await click('Galaxy'); await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="galaxy"')); await delay(300)
      const layout = await evaluate('(()=>{const box=e=>{const r=e.getBoundingClientRect();return {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}};return {sidebar:box(document.querySelector(\'.go-galaxy-sidebar\')),area:box(document.querySelector(\'.go-galaxy-area\')),planes:Array.from(document.querySelectorAll(\'.go-galaxy-plane>svg\')).map(box),controls:Array.from(document.querySelectorAll(\'.go-galaxy-sidebar button\')).map(box),header:!!document.querySelector(\'header\'),navigation:!!document.querySelector(\'.secondary-navigation\'),horizontalOverflow:document.querySelector(\'.go-galaxy-sidebar\').scrollWidth>document.querySelector(\'.go-galaxy-sidebar\').clientWidth}})()')
      if (layout.sidebar.right > layout.area.left + 1 || layout.planes.length !== 3 || layout.planes.some(p => p.right > width + 1 || p.bottom > height + 1) || layout.controls.some(p => p.height < 44) || layout.horizontalOverflow || !layout.header || !layout.navigation) throw Error('Touch layout regression ' + name + JSON.stringify(layout))
      const galaxyGeometry = checkGeometry(await geometry('.go-terminal .go-g3d'), 'galaxy ' + name)
      report.layouts.push({ name, ...layout, geometry: galaxyGeometry }); await shot('galaxy-' + name)
      await evaluate('window.g3eLocal=true'); await click('Local'); await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="local"'))
      await shot('local-before-adjustment-' + name)
      if (!sameRects(checkGeometry(await geometry('.go-terminal .go-g3d'), 'local ' + name), galaxyGeometry)) throw Error('Galaxy and Local panel geometry differ ' + name)
      if (await evaluate('document.querySelectorAll(".go-galaxy-axes button").length') !== 6) throw Error('XYZ controls missing')
      await evaluate('document.querySelector(\'[aria-label="Increase X extent"]\').click()')
      await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
      const unchanged = await evaluate('Array.from(document.querySelectorAll(".go-galaxy-plane>svg")).map(e=>{const r=e.getBoundingClientRect();return {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}})')
      if (unchanged.some((r, i) => Object.keys(r).some(k => Math.abs(r[k] - layout.planes[i][k]) > 1))) throw Error('XYZ or mode changed panel rectangles')
      const beforePositions = await evaluate('Array.from(document.querySelectorAll("[data-marker=destination][data-axes=xz]")).map(e=>e.dataset.position).join(";")')
      for (const axis of ['Y', 'Z']) { await evaluate(`document.querySelector('[aria-label="Increase ${axis} extent"]').click()`); await delay(250) }
      const resized = await evaluate('Array.from(document.querySelectorAll(".go-galaxy-plane>svg")).map(e=>{const r=e.getBoundingClientRect();return {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}})')
      if (resized.some((r, i) => Object.keys(r).some(k => Math.abs(r[k] - layout.planes[i][k]) > 1))) throw Error('YZ changed panel rectangles')
      const afterPositions = await evaluate('Array.from(document.querySelectorAll("[data-marker=destination][data-axes=xz]")).map(e=>e.dataset.position).join(";")')
      if (beforePositions !== afterPositions) throw Error('XYZ slice edit unexpectedly moved the fixed X/Z camera')
      report.layouts.at(-1).stableXYZ = true; report.layouts.at(-1).sliceEditingKeepsCamera = true
      await shot('local-adjusted-' + name)
      await click('Fit Route'); await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("200 LY")'))
    }
    // Physical zoom transitions retain the same inspection centre.
    for (let i = 0; i < 3; i++) { await evaluate('document.querySelector(\'[aria-label="Galaxy zoom out"]\').click()'); await delay(50) }
    await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="galaxy"'))
    await evaluate('document.querySelector(\'[aria-label="Galaxy zoom in"]\').click()')
    await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="local"'))
    report.zoomTransitions = true
    // The XYZ controls now stay visible while Local data load (Task 3 / F04),
    // so wait for the fitted slice's data, not only its axis text, before
    // counting idle requests.
    await click('Fit Route'); await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("200 LY")&&document.querySelector(".go-g3d").dataset.mode==="local"&&!document.querySelector(".go-galaxy-local-status")'))
    const count = await evaluate('window.g3eRequests.length'); const first = Object.fromEntries((await c.call('Performance.getMetrics')).metrics.map(m => [m.name, m.value])); await delay(1200)
    const last = Object.fromEntries((await c.call('Performance.getMetrics')).metrics.map(m => [m.name, m.value])); report.idle = { heapBytes: last.JSHeapUsedSize, mainThreadPercent: 100 * (last.TaskDuration - first.TaskDuration) / (last.Timestamp - first.Timestamp) }
    if (await evaluate('window.g3eRequests.length') !== count) throw Error('Stable view repeatedly requested spatial data')
    // Task 3 / F03: FSD charge and jump keep the canonical geometry at every
    // size, with a route; an XYZ edit inside the overlay keeps the rectangles;
    // View System leaves the overlay through the manual dismissal policy.
    report.fsd = []
    for (const [name, width, height] of [['laptop', 1366, 768], ['desktop', 1920, 1080], ['touch', 820, 1180], ['compact', 680, 580]]) {
      await c.call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false })
      await c.call('Page.navigate', { url: origin + '/nav/galaxy' }); await waitFor(() => evaluate(pageTextReady('Galaxy Map', true))); await evaluate('window.g3eLocal=true')
      // A charge is only observed in a live session: arrive first.
      // Wait until the host has read that journal line, so the charge Status
      // is not observed before (and then cleared by) the arrival.
      const eventsBefore = (await rpc('getJournalSummary')).eventsRead
      host.child.stdin.write('arrival\n')
      await waitFor(async () => (await rpc('getJournalSummary')).eventsRead > eventsBefore)
      await waitFor(async () => { const n = (await rpc('getGoNavigationContext'))?.navigation; return n?.fresh && !['charge', 'jump'].includes(n.mode) })
      host.child.stdin.write('charge\n')
      await waitFor(() => evaluate(`!!document.querySelector(${JSON.stringify(fsdScope)})&&document.querySelector('[aria-label="Automatic FSD display"] h2')?.textContent==='FSD charging'`))
      const charge = checkGeometry(await geometry(fsdScope), 'FSD charge route ' + name); await shot('fsd-charge-' + name)
      host.child.stdin.write('jump\n')
      await waitFor(() => evaluate('document.querySelector(\'[aria-label="Automatic FSD display"] h2\')?.textContent==="Hyperspace jump"'))
      const jump = checkGeometry(await geometry(fsdScope), 'FSD jump route ' + name); await shot('fsd-jump-' + name)
      const entry = { name, charge, jump }
      if (name === 'laptop') {
        await click('Local'); await waitFor(() => evaluate(`document.querySelector(${JSON.stringify(fsdScope)}).dataset.mode==='local'`))
        const before = checkGeometry(await geometry(fsdScope), 'FSD local ' + name)
        await evaluate(`document.querySelector(${JSON.stringify(fsdScope)}).querySelector('[aria-label="Increase X extent"]').click()`)
        await waitFor(async () => (await geometry(fsdScope))?.mode === 'local' && await evaluate(`document.querySelector(${JSON.stringify(fsdScope)}).querySelector('.go-galaxy-axes')?.textContent.includes('LY')`))
        if (!sameRects(checkGeometry(await geometry(fsdScope), 'FSD local edited ' + name), before)) throw Error('XYZ edit changed FSD panel rectangles')
        entry.xyzEditKeepsRects = true
        // Search inside the FSD overlay, then View System leaves the overlay
        // through the manual dismissal policy.
        await setQuery('Fixture Alpha')
        await waitFor(() => evaluate('!document.querySelector(\'[aria-label="Automatic FSD display"] .go-galaxy-sidebar form button\').disabled'))
        await evaluate('document.querySelector(\'[aria-label="Automatic FSD display"] .go-galaxy-sidebar form button\').click()')
        await waitFor(() => evaluate('document.querySelectorAll(\'[aria-label="Automatic FSD display"] [data-marker=searched]\').length===3'))
        await waitFor(() => evaluate('Array.from(document.querySelectorAll(\'[aria-label="Automatic FSD display"] button\')).some(b=>b.textContent===\'View System\')'))
        await evaluate('Array.from(document.querySelectorAll(\'[aria-label="Automatic FSD display"] button\')).find(b=>b.textContent===\'View System\').click()')
        await waitFor(() => evaluate('location.pathname==="/nav/map"&&!document.querySelector(\'[aria-label="Automatic FSD display"]\')'))
        entry.viewSystemLeavesFSD = true; await shot('fsd-view-system-' + name)
      } else {
        await click('Return to previous screen')
        await waitFor(() => evaluate('!document.querySelector(\'[aria-label="Automatic FSD display"]\')'))
      }
      host.child.stdin.write('arrival\n')
      await waitFor(async () => !['charge', 'jump'].includes(await navMode()))
      report.fsd.push(entry)
    }
    await c.call('Page.navigate', { url: origin + '/nav/galaxy' }); await waitFor(() => evaluate(pageTextReady('Galaxy Map', true)))
    host.child.stdin.write('clear-route\n'); await waitFor(async () => !(await rpc('getGoNavigation')).hops.length)
    await click('Local'); await waitFor(() => evaluate('!!document.querySelector(".go-galaxy-axes")'))
    await evaluate('document.querySelector(\'[aria-label="Increase X extent"]\').click()')
    await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
    host.child.stdin.write('jump\n')
    await waitFor(() => evaluate('!!document.querySelector(\'[aria-label="Automatic FSD display"] .go-g3d\')'))
    report.fsdNoRoute = checkGeometry(await geometry(fsdScope), 'FSD jump no route'); await shot('fsd-jump-no-route')
    await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
    await click('Return to previous screen'); await delay(1200)
    if (!await evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")')) throw Error('Manual inspection lost through FSD overlay')
    if (await evaluate('!!document.querySelector(\'[aria-label="Automatic FSD display"]\')')) throw Error('Dismissed episode reopened')
    host.child.stdin.write('arrival\n'); await delay(1200)
    await rpc('setGoCompanion', { ...config, galaxyMap: false, localAtlas: false })
    await waitFor(() => evaluate('document.body.textContent.includes("Map renderers disabled")'))
    const disabledCount = await evaluate('window.g3eRequests.length'); await delay(1000)
    if (await evaluate('window.g3eRequests.length') !== disabledCount) throw Error('Disabled spatial renderer queried')
    report.stableRequests = count; report.fsdDismissal = true; report.noRouteFSD = true; report.inspectionSurvivesFSD = true; report.bothDisabled = true
    if (report.errors.length || report.external.length) throw Error('Browser error or external traffic')
    report.passed = true; console.log(JSON.stringify(report))
  } finally { report.diagnostic = await evaluate('({local:window.g3eLocal,requests:window.g3eRequests,text:document.body.innerText,mode:document.querySelector(".go-g3d")?.dataset.mode})').catch(() => null); if (host) await host.stop(); await b.stop(); await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2)) }
}
if (require.main === module) main().catch(e => { console.error(e); process.exitCode = 1 })
