const CACHE_NAME = 'leo-concentration-v1';
const ASSETS = [
  '/app/index.html',
  '/app/manifest.json',
  '/static/app.css'
];
self.addEventListener('install', (e)=>{
  e.waitUntil(caches.open(CACHE_NAME).then(c=>c.addAll(ASSETS)));
});
self.addEventListener('activate', (e)=>{
  e.waitUntil(caches.keys().then(keys=>Promise.all(keys.map(k=>{ if(k!==CACHE_NAME) return caches.delete(k); }))))
});
self.addEventListener('fetch', (e)=>{
  const req = e.request;
  e.respondWith(
    caches.match(req).then(cached=> cached || fetch(req).then(res=>{
      // cache GET requests
      if(req.method==='GET' && res.status===200 && (req.url.includes('/static/') || req.url.includes('/app/'))){
        const resClone = res.clone(); caches.open(CACHE_NAME).then(c=>c.put(req, resClone));
      }
      return res;
    }).catch(()=> caches.match('/app/index.html')))
  );
});

