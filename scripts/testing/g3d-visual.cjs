const fs = require('node:fs/promises')
const path = require('node:path')
const { fixture, browser, waitFor, delay, pageTextReady } = require('./g2d-visual.cjs')
const root = path.resolve(__dirname, '../..')
async function main () {
  const output = path.join(root, 'temp/g3d-visual'); await fs.mkdir(output, { recursive: true })
  const report = { scope: 'Native loopback UI; synthetic journal and explicitly injected spatial response, not gameplay or publisher validation', errors: [], external: [], layouts: [] }
  const b = await browser(output); const c = b.client; let host; let origin = ''
  const evaluate = async expression => { const r = await c.call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }); if (r.exceptionDetails) throw Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text); return r.result.value }
  const click = text => evaluate(`Array.from(document.querySelectorAll('.go-galaxy-sidebar button, .go-fsd>.go-page__heading button')).find(b=>b.textContent===${JSON.stringify(text)})?.click()`)
  const shot = async name => { const r = await c.call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false }); await fs.writeFile(path.join(output, name + '.png'), Buffer.from(r.data, 'base64')) }
  const rpc = (name, message) => evaluate(`new Promise((resolve,reject)=>{const w=new WebSocket(location.origin.replace('http:','ws:')+'/?scope=launcher');w.onopen=()=>w.send(JSON.stringify({requestId:'g3d',name:${JSON.stringify(name)},message:${JSON.stringify(message)}}));w.onmessage=e=>{const r=JSON.parse(e.data);if(r.requestId==='g3d'){w.close();r.error?reject(Error(r.error)):resolve(r.message)}};w.onerror=reject})`)
  c.on(e => {
    if (e.method === 'Runtime.exceptionThrown') report.errors.push(e.params.exceptionDetails.exception?.description)
    if (e.method === 'Fetch.requestPaused') { const p = e.params; const ok = p.request.url.startsWith(origin + '/') || p.request.url.startsWith('data:'); if (!ok) report.external.push(p.request.url); c.call(ok ? 'Fetch.continueRequest' : 'Fetch.failRequest', ok ? { requestId: p.requestId } : { requestId: p.requestId, errorReason: 'BlockedByClient' }).catch(() => {}) }
  })
  await c.call('Network.enable'); await c.call('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  // Only the browser spatial-view response is synthetic. All owner settings,
  // navigation, FSD episodes, permissions and lifecycle use the real Go fixture.
  await c.call('Page.addScriptToEvaluateOnNewDocument', { source: 'window.g3dRequests=[];const send=WebSocket.prototype.send;WebSocket.prototype.send=function(raw){let q;try{q=JSON.parse(raw)}catch{};if(q?.name===\'getGoSpatialView\'){window.g3dRequests.push(q);if(window.g3dLocal){queueMicrotask(()=>this.onmessage?.({data:JSON.stringify({requestId:q.requestId,message:{complete:true,pending:false,points:[[3,4,5],[8,4,5]],notice:\'SYNTHETIC UI FIXTURE\'}})}));return}}return send.call(this,raw)}' })
  try {
    host = await fixture('G2E', '', path.join(root, 'tests/testdata/g2g')); origin = host.url
    await c.call('Page.navigate', { url: origin + '/nav/galaxy' }); await waitFor(() => evaluate(pageTextReady('Galaxy Map', true)))
    const config = { charge: false, jump: false, restore: true, owner: true, allViewers: false, devices: [], closeMode: 'tray', galaxyMap: true, localAtlas: true, mapDuringFSD: true }
    await rpc('setGoCompanion', config)
    for (const [name, width, height] of [['desktop', 1920, 1080], ['laptop', 1366, 768], ['touch', 820, 1180], ['compact', 680, 580]]) {
      await c.call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false })
      await evaluate('document.fonts.ready.then(()=>true)'); await click('Galaxy'); await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="galaxy"')); await delay(300)
      const layout = await evaluate('(()=>{const box=e=>{const r=e.getBoundingClientRect();return {left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width,height:r.height}};return {sidebar:box(document.querySelector(\'.go-galaxy-sidebar\')),area:box(document.querySelector(\'.go-galaxy-area\')),planes:Array.from(document.querySelectorAll(\'.go-galaxy-plane>svg\')).map(box),controls:Array.from(document.querySelectorAll(\'.go-galaxy-sidebar button\')).map(box),header:!!document.querySelector(\'header\'),navigation:!!document.querySelector(\'.secondary-navigation\'),horizontalOverflow:document.querySelector(\'.go-galaxy-sidebar\').scrollWidth>document.querySelector(\'.go-galaxy-sidebar\').clientWidth}})()')
      if (layout.sidebar.right > layout.area.left + 1 || layout.planes.length !== 3 || layout.planes.some(p => p.right > width + 1 || p.bottom > height + 1) || layout.controls.some(p => p.height < 44) || layout.horizontalOverflow || !layout.header || !layout.navigation) throw Error('Touch layout regression ' + name + JSON.stringify(layout))
      report.layouts.push({ name, ...layout }); await shot('galaxy-' + name)
      await evaluate('window.g3dLocal=true'); await click('Local'); await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="local"'))
      if (await evaluate('document.querySelectorAll(".go-galaxy-axes button").length') !== 6) throw Error('XYZ controls missing')
      await evaluate('document.querySelector(\'[aria-label="Increase X extent"]\').click()')
      await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
      await shot('local-' + name)
      await click('Fit Route'); await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("200 LY")'))
    }
    // Physical zoom transitions retain the same inspection centre.
    for (let i = 0; i < 3; i++) { await evaluate('document.querySelector(\'[aria-label="Galaxy zoom out"]\').click()'); await delay(50) }
    await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="galaxy"'))
    await evaluate('document.querySelector(\'[aria-label="Galaxy zoom in"]\').click()')
    await waitFor(() => evaluate('document.querySelector(".go-g3d").dataset.mode==="local"'))
    report.zoomTransitions = true
    // XYZ controls stay visible while Local data load (Task 3 / F04): wait for
    // the fitted slice's data before counting idle requests.
    await click('Fit Route'); await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("200 LY")&&document.querySelector(".go-g3d").dataset.mode==="local"&&!document.querySelector(".go-galaxy-local-status")'))
    const count = await evaluate('window.g3dRequests.length'); const first = Object.fromEntries((await c.call('Performance.getMetrics')).metrics.map(m => [m.name, m.value])); await delay(1200)
    const last = Object.fromEntries((await c.call('Performance.getMetrics')).metrics.map(m => [m.name, m.value])); report.idle = { heapBytes: last.JSHeapUsedSize, mainThreadPercent: 100 * (last.TaskDuration - first.TaskDuration) / (last.Timestamp - first.Timestamp) }
    if (await evaluate('window.g3dRequests.length') !== count) throw Error('Stable view repeatedly requested spatial data')
    host.child.stdin.write('clear-route\n'); await waitFor(async () => !(await rpc('getGoNavigation')).hops.length)
    await click('Local'); await waitFor(() => evaluate('!!document.querySelector(".go-galaxy-axes")'))
    await evaluate('document.querySelector(\'[aria-label="Increase X extent"]\').click()')
    await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
    host.child.stdin.write('jump\n')
    await waitFor(() => evaluate('!!document.querySelector(\'[aria-label="Automatic FSD display"] .go-g3d\')'))
    await waitFor(() => evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")'))
    await click('Return to previous screen'); await delay(1200)
    if (!await evaluate('document.querySelector(".go-galaxy-axes")?.textContent.includes("250 LY")')) throw Error('Manual inspection lost through FSD overlay')
    if (await evaluate('!!document.querySelector(\'[aria-label="Automatic FSD display"]\')')) throw Error('Dismissed episode reopened')
    host.child.stdin.write('arrival\n'); await delay(1200)
    await rpc('setGoCompanion', { ...config, galaxyMap: false, localAtlas: false })
    await waitFor(() => evaluate('document.body.textContent.includes("Map renderers disabled")'))
    const disabledCount = await evaluate('window.g3dRequests.length'); await delay(1000)
    if (await evaluate('window.g3dRequests.length') !== disabledCount) throw Error('Disabled spatial renderer queried')
    report.stableRequests = count; report.fsdDismissal = true; report.noRouteFSD = true; report.inspectionSurvivesFSD = true; report.bothDisabled = true
    if (report.errors.length || report.external.length) throw Error('Browser error or external traffic')
    report.passed = true; console.log(JSON.stringify(report))
  } finally { report.diagnostic = await evaluate('({local:window.g3dLocal,requests:window.g3dRequests,text:document.body.innerText,mode:document.querySelector(".go-g3d")?.dataset.mode})').catch(() => null); if (host) await host.stop(); await b.stop(); await fs.writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2)) }
}
if (require.main === module) main().catch(e => { console.error(e); process.exitCode = 1 })
