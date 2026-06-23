// Maps CoreDNS plugins to a coarse "what does it do to the query" category, used
// to give each request-flow step a friendly icon. Unknown/unlisted plugins fall
// back to a generic icon.

/** @type {Record<string, string>} category → emoji icon */
const CATEGORY_ICON = {
  observe: '📝', // logging / metrics / tracing
  health: '🩺', // health & readiness endpoints
  cache: '📦', // caching
  answer: '🎯', // produces the answer from local data
  forward: '➡️', // sends the query upstream
  security: '🔒', // access control, DNSSEC, TLS, TSIG
  transform: '✏️', // rewrites / reshapes the query or response
  serverctl: '⚙️', // server/runtime configuration
}

/** @type {Record<string, keyof typeof CATEGORY_ICON>} plugin name → category */
const PLUGIN_CATEGORY = {
  // observe
  log: 'observe', errors: 'observe', trace: 'observe', dnstap: 'observe',
  prometheus: 'observe', debug: 'observe', nsid: 'observe',
  // health
  health: 'health', ready: 'health', pprof: 'health',
  // cache
  cache: 'cache',
  // answer (resolve from local/cluster data, or test responders)
  file: 'answer', auto: 'answer', secondary: 'answer', hosts: 'answer',
  kubernetes: 'answer', etcd: 'answer', route53: 'answer', azure: 'answer',
  clouddns: 'answer', k8s_external: 'answer', template: 'answer', whoami: 'answer',
  erratic: 'answer', chaos: 'answer', any: 'answer', local: 'answer',
  // forward
  forward: 'forward', grpc: 'forward', dns64: 'forward',
  // security
  acl: 'security', dnssec: 'security', tsig: 'security', tls: 'security',
  sign: 'security', bufsize: 'security',
  // transform
  rewrite: 'transform', header: 'transform', minimal: 'transform',
  loadbalance: 'transform', autopath: 'transform',
  // serverctl
  bind: 'serverctl', view: 'serverctl', cancel: 'serverctl', timeouts: 'serverctl',
  multisocket: 'serverctl', reload: 'serverctl', loop: 'serverctl', root: 'serverctl',
  metadata: 'serverctl', geoip: 'serverctl', transfer: 'serverctl', on: 'serverctl',
}

/**
 * Returns an emoji icon for a plugin's category, or a generic icon when the
 * plugin is unknown/unlisted.
 * @param {string} name
 * @returns {string}
 */
export function pluginIcon(name) {
  const category = PLUGIN_CATEGORY[name]
  return (category && CATEGORY_ICON[category]) || '🔌'
}
