import assert from 'node:assert/strict'
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import ts from 'typescript'
import { createPinia, setActivePinia } from 'pinia'
import { AxiosError } from 'axios'

// Exercise the real store and interceptor with a fresh Pinia instance per page load.
const root = dirname(dirname(fileURLToPath(import.meta.url)))
const cache = join(root, 'node_modules', '.cache')
await mkdir(cache, { recursive: true })
const directory = await mkdtemp(join(cache, 'auth-session-'))
try {
  for (const [source, output] of [
    ['src/api/client.ts', 'client.mjs'],
    ['src/api/auth.ts', 'auth.mjs'],
    ['src/stores/auth.ts', 'store.mjs']
  ]) {
    const content = (await readFile(join(root, source), 'utf8'))
      .replace("from './client'", "from './client.mjs'")
      .replace("from '@/api/auth'", "from './auth.mjs'")
    await writeFile(join(directory, output), ts.transpileModule(content, {
      compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 }
    }).outputText)
  }
  const { default: client } = await import(pathToFileURL(join(directory, 'client.mjs')))
  const { useAuthStore } = await import(pathToFileURL(join(directory, 'store.mjs')))
  const user = { id: 1, email: 'session@example.test', role: 'admin' }
  let validCookie = true
  let setupCompleted = true
  const requests = []
  globalThis.window = { location: { href: '/devices' } }
  client.defaults.adapter = async config => {
    requests.push(config.url)
    assert.equal(config.withCredentials, true)
    if (config.url === '/me' && !validCookie || config.url === '/auth/login') {
      throw new AxiosError('Unauthenticated', 'ERR_BAD_REQUEST', config, undefined, {
        status: 401, data: { error: { message: 'Invalid credentials' } }, config
      })
    }
    if (config.url === '/auth/logout') validCookie = false
    return {
      status: 200, statusText: 'OK', headers: {}, config,
      data: config.url === '/setup/status' ? { setup_completed: setupCompleted } : user
    }
  }
  const freshPage = () => { setActivePinia(createPinia()); return useAuthStore() }
  let store = freshPage()
  await Promise.all([store.checkAuth(), store.checkAuth()])
  assert.equal(store.isAuthenticated, true)
  assert.equal(store.user.email, user.email)
  assert.equal(requests.filter(url => url === '/me').length, 1)
  store = freshPage() // A refresh clears all frontend memory, but retains the cookie.
  await store.checkAuth()
  assert.equal(store.isAuthenticated, true)
  await store.logout()
  assert.equal(store.isAuthenticated, false)
  store = freshPage()
  await store.checkAuth()
  assert.equal(store.isAuthenticated, false)
  assert.equal(window.location.href, '/devices', 'session check must not reload the login page')
  assert.equal(await store.login('invalid@example.test', 'wrong-password'), false)
  assert.equal(store.error, 'Invalid credentials')
  assert.equal(window.location.href, '/devices')
  setupCompleted = false
  requests.length = 0
  store = freshPage()
  await store.checkAuth()
  assert.equal(store.setupCompleted, false)
  assert.deepEqual(requests, ['/setup/status'])
  console.log('Cookie session restore, refresh, concurrent checks, logout, expired session and failed login passed')
} finally {
  await rm(directory, { recursive: true, force: true })
}
