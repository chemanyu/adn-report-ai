// Run with Node >=18 after installing Playwright in .cache/browser.
// The separate instance at :18081 uses an isolated, disposable PostgreSQL schema.
const { chromium } = require('../.cache/browser/node_modules/playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');

(async () => {
 const base = (process.env.ADN_E2E_BASE_URL || 'http://127.0.0.1:18081').replace(/\/+$/, '');
 const output = process.env.ADN_E2E_OUTPUT || '.cache/e2e';
 fs.mkdirSync(output, { recursive: true });
 const configPath = process.env.ADN_E2E_CONFIG || '.cache/e2e/config.yaml';
 const config = fs.readFileSync(configPath, 'utf8');
 const password = configPath.endsWith('.json') ? JSON.parse(config).Auth.AdminPassword : config.match(/AdminPassword:\s*"([^"]+)"/)[1];
 const browser = await chromium.launch({executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});
 try {
  const context = await browser.newContext({viewport:{width:1440,height:1000},locale:'zh-CN'});
  const page = await context.newPage();
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto(base);
  await page.locator('#login-view').waitFor({state:'visible'});
  await page.screenshot({path:path.join(output, 'login.png'),fullPage:true});
  await page.locator('#local-section summary').click();
  await page.locator('#password').fill(password);
  await page.locator('#login-form button').click();
  await page.locator('#app-view').waitFor({state:'visible'});
  await page.locator('#empty-records').waitFor({state:'visible'});
  await page.screenshot({path:path.join(output, 'workspace-empty.png'),fullPage:true});
  await page.locator('#nav-reports').click();
  const template = await context.request.get(base+'/api/template');
  assert.equal(template.status(),200);
  const templatePath=path.join(output, 'template.xlsx');fs.writeFileSync(templatePath,await template.body());
  assert.equal(await page.locator('#nav-accounts').count(),0);
  await page.locator('#open-upload').click();
  assert.equal(await page.locator('#upload-account').count(),0);
  await page.locator('#file').setInputFiles(templatePath);
  await page.locator('#validation .success').waitFor({state:'visible'});
  assert.equal(await page.locator('#operator').inputValue(),'管理员');
  await page.locator('#operator').fill('验收运营');
  await page.screenshot({path:path.join(output, 'upload.png'),fullPage:true});
  await page.locator('#submit-upload').click();
  await page.locator('#upload-dialog').waitFor({state:'hidden'});
  await page.locator('#stat-files').filter({hasText:/^1$/}).waitFor({state:'visible'});
  assert.equal(await page.locator('#stat-amount').textContent(),'30.00');
  assert.equal(await page.locator('#stat-advertisers').textContent(),'1');
  assert.match(await page.locator('#records').textContent(),/验收运营/);
  await page.screenshot({path:path.join(output, 'workspace.png'),fullPage:true});
  await page.locator('[data-detail]').click();
  await page.locator('#detail-table tbody tr').waitFor({state:'visible'});
  assert.match(await page.locator('#detail-table').textContent(),/示例广告主/);
  await page.screenshot({path:path.join(output, 'details.png'),fullPage:true});
  const csv=await context.request.get(new URL(await page.locator('#detail-download').getAttribute('href'), base+'/').href);
  assert.equal(csv.status(),200);assert.match(await csv.text(),/示例广告主/);
  await page.locator('[data-close="detail-dialog"]').click();
  // Verify invalid workbook content is rejected without relying on local sample files.
  await page.locator('#open-upload').click();
  await page.locator('#file').setInputFiles({name:'invalid.xlsx',mimeType:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',buffer:Buffer.from('invalid workbook')});
  await page.locator('#validation .error').waitFor({state:'visible'});
  assert.match(await page.locator('#validation').textContent(),/无法读取 Excel/);
  assert.equal(await page.locator('#submit-upload').isDisabled(),true);
  await page.screenshot({path:path.join(output, 'validation.png'),fullPage:true});
  await page.locator('[data-close="upload-dialog"]').first().click();
  await page.setViewportSize({width:390,height:844});
  await page.screenshot({path:path.join(output, 'mobile.png'),fullPage:true});
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
  assert.equal(overflow,false,'mobile page should not overflow horizontally');
  await page.locator('#logout-mobile').click();
  await page.locator('#login-view').waitFor({state:'visible'});
  assert.deepEqual(errors,[]);
  console.log('PASS: browser login, Excel preview/upload, totals, detail, CSV, validation, mobile layout and logout; no JavaScript errors.');
 } finally { await browser.close(); }
})().catch(error=>{console.error(error);process.exitCode=1;});
