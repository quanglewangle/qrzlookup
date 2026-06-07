package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/quanglewangle/qrzlook/db"
	"github.com/quanglewangle/qrzlook/qrz"
	"github.com/quanglewangle/qrzlook/terrain50"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ROC Locations</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
<link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.css"/>
<link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.Default.css"/>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script src="https://unpkg.com/leaflet.markercluster@1.5.3/dist/leaflet.markercluster.js"></script>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f4f8; color: #1a202c; min-height: 100vh; }

  nav {
    background: #2b6cb0;
    color: white;
    padding: 0 1.5rem;
    display: flex;
    align-items: center;
    gap: 0;
    height: 52px;
  }
  nav .brand { font-weight: 700; font-size: 1.1rem; margin-right: 1.5rem; }
  nav a {
    color: rgba(255,255,255,0.8);
    text-decoration: none;
    padding: 0 1rem;
    height: 52px;
    line-height: 52px;
    font-size: 0.95rem;
    transition: background 0.15s;
  }
  nav a:hover, nav a.active { background: rgba(0,0,0,0.2); color: white; }

  .view { display: none; padding: 2rem 1rem; }
  .view.active { display: block; }

  /* Lookup view */
  #view-lookup { max-width: 520px; margin: 0 auto; }
  #view-lookup h2 { font-size: 1.3rem; color: #2d3748; margin-bottom: 1.25rem; }
  .search-form { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; }
  .search-form input {
    flex: 1; padding: 0.7rem 1rem; font-size: 1.05rem;
    border: 2px solid #cbd5e0; border-radius: 8px; outline: none;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .search-form input:focus { border-color: #4299e1; }
  .btn { padding: 0.7rem 1.25rem; font-size: 0.95rem; font-weight: 600; border: none; border-radius: 8px; cursor: pointer; transition: background 0.15s; }
  .btn-primary { background: #3182ce; color: white; }
  .btn-primary:hover { background: #2b6cb0; }
  .btn-primary:disabled { background: #a0aec0; cursor: not-allowed; }
  .btn-success { background: #38a169; color: white; }
  .btn-success:hover { background: #2f855a; }
  .btn-danger { background: #e53e3e; color: white; }
  .btn-danger:hover { background: #c53030; }
  .btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }

  .card { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 12px rgba(0,0,0,0.08); margin-bottom: 1rem; }
  .callsign-big { font-size: 2rem; font-weight: 800; color: #2b6cb0; letter-spacing: 0.05em; }
  .name-big { font-size: 1.15rem; color: #2d3748; font-weight: 500; margin: 0.2rem 0 1rem; }
  .divider { border: none; border-top: 1px solid #e2e8f0; margin: 1rem 0; }
  .fields { display: flex; flex-direction: column; gap: 0.55rem; }
  .field { display: flex; justify-content: space-between; gap: 1rem; }
  .field-label { font-size: 0.78rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: #a0aec0; }
  .field-value { font-size: 0.92rem; color: #2d3748; text-align: right; }
  .map-link { display: inline-block; margin-top: 1rem; font-size: 0.85rem; color: #3182ce; text-decoration: none; }
  .map-link:hover { text-decoration: underline; }
  .error-box { background: #fff5f5; border: 1px solid #fed7d7; color: #c53030; padding: 1rem 1.25rem; border-radius: 8px; font-size: 0.92rem; }
  .spinner { width: 32px; height: 32px; border: 3px solid #e2e8f0; border-top-color: #3182ce; border-radius: 50%; animation: spin 0.7s linear infinite; margin: 1.5rem auto; }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* Sites view */
  #view-sites { max-width: 800px; margin: 0 auto; }
  .sites-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  .sites-header h2 { font-size: 1.3rem; color: #2d3748; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08); font-size: 0.88rem; }
  th { background: #edf2f7; color: #718096; font-size: 0.72rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; padding: 0.65rem 1rem; text-align: left; }
  td { padding: 0.6rem 1rem; border-top: 1px solid #f0f4f8; color: #2d3748; }
  tr:hover td { background: #f7fafc; }
  .cs-link { font-weight: 700; color: #2b6cb0; cursor: pointer; }
  .cs-link:hover { text-decoration: underline; }
  .actions { display: flex; gap: 0.4rem; }

  /* Map view */
  #view-map { padding: 0; }
  #map { height: calc(100vh - 52px); width: 100%; }

  /* Modal */
  .modal-overlay { display: none; position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center; }
  .modal-overlay.open { display: flex; }
  .modal-box { background: white; border-radius: 12px; padding: 1.75rem; width: 100%; max-width: 420px; margin: 1rem; box-shadow: 0 8px 32px rgba(0,0,0,0.2); }
  .modal-box h3 { font-size: 1.15rem; margin-bottom: 1.25rem; color: #2d3748; }
  .form-group { margin-bottom: 1rem; }
  .form-group label { display: block; font-size: 0.8rem; font-weight: 600; color: #4a5568; margin-bottom: 0.3rem; text-transform: uppercase; letter-spacing: 0.06em; }
  .form-group input { width: 100%; padding: 0.6rem 0.8rem; border: 2px solid #cbd5e0; border-radius: 8px; font-size: 0.95rem; outline: none; }
  .form-group input:focus { border-color: #4299e1; }
  .modal-footer { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1.25rem; }
  .btn-ghost { background: #e2e8f0; color: #4a5568; }
  .btn-ghost:hover { background: #cbd5e0; }
</style>
</head>
<body>

<nav>
  <span class="brand">ROC Locations</span>
  <a href="#" class="active" data-view="lookup" title="Look up and edit">Lookup</a>
  <a href="#" data-view="sites">Sites</a>
  <a href="#" data-view="map">Map</a>
</nav>

<div id="view-lookup" class="view active">
  <h2>Callsign Lookup</h2>
  <form class="search-form" id="lookup-form">
    <input type="text" id="cs-input" placeholder="e.g. G8GDS" autocomplete="off" autocorrect="off" spellcheck="false">
    <button class="btn btn-primary" id="lookup-btn" type="submit">Look up</button>
  </form>
  <div id="lookup-result"></div>
</div>

<div id="view-sites" class="view">
  <div class="sites-header">
    <h2>ROC Sites</h2>
    <button class="btn btn-success" onclick="openAddModal()">+ Add New</button>
  </div>
  <table>
    <thead><tr><th>Callsign</th><th>Name</th><th>Lat</th><th>Lon</th><th title="Antenna height above ground (m)">QNF</th><th title="Height above sea level (m)">QNH (m)</th><th></th></tr></thead>
    <tbody id="sites-tbody"></tbody>
  </table>
</div>

<div id="view-map" class="view">
  <div id="map"></div>
</div>

<div class="modal-overlay" id="modal">
  <div class="modal-box">
    <h3 id="modal-title">Add Site</h3>
    <form id="site-form">
      <div class="form-group">
        <label>Callsign</label>
        <input type="text" id="f-callsign" required style="text-transform:uppercase">
        <button type="button" class="btn btn-primary btn-sm" style="margin-top:0.4rem" onclick="qrzFill()">Fill from QRZ</button>
      </div>
      <div class="form-group"><label>Name</label><input type="text" id="f-name"></div>
      <div class="form-group"><label>Latitude</label><input type="number" id="f-lat" step="any"></div>
      <div class="form-group"><label>Longitude</label><input type="number" id="f-lon" step="any"></div>
      <div class="form-group"><label title="Antenna height above ground (m)">QNF</label><input type="number" id="f-qnf" step="any" value="3"></div>
      <div class="form-group"><label title="Height above sea level (m)">QNH (m)</label><input type="number" id="f-qnh" step="any" placeholder="from QRZ"></div>
      <div class="modal-footer">
        <button type="button" class="btn btn-ghost" onclick="closeModal()">Cancel</button>
        <button type="submit" class="btn btn-primary" id="modal-save">Save</button>
      </div>
    </form>
  </div>
</div>

<script>
// ── Navigation ───────────────────────────────────────────────
let leafletMap = null;

document.querySelectorAll('nav a').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault();
    const view = a.dataset.view;
    document.querySelectorAll('nav a').forEach(x => x.classList.remove('active'));
    a.classList.add('active');
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    document.getElementById('view-' + view).classList.add('active');
    if (view === 'sites') loadSitesTable();
    if (view === 'map') initMap();
  });
});

// ── Lookup ───────────────────────────────────────────────────
document.getElementById('lookup-form').addEventListener('submit', async e => {
  e.preventDefault();
  const cs = document.getElementById('cs-input').value.trim().toUpperCase();
  if (!cs) return;
  const btn = document.getElementById('lookup-btn');
  const res = document.getElementById('lookup-result');
  btn.disabled = true;
  res.innerHTML = '<div class="spinner"></div>';
  try {
    const data = await fetch('/qrz/lookup/' + encodeURIComponent(cs)).then(r => r.json());
    if (data.error) {
      res.innerHTML = '<div class="error-box">' + esc(data.error) + '</div>';
    } else {
      const loc = [data.city, data.state, data.country].filter(Boolean).join(', ');
      const mapURL = data.lat && data.lon
        ? 'https://www.openstreetmap.org/?mlat=' + data.lat + '&mlon=' + data.lon + '&zoom=10' : null;
      res.innerHTML = '<div class="card">' +
        '<div class="callsign-big">' + esc(data.callsign) + '</div>' +
        '<div class="name-big">' + esc(data.name || '—') + '</div>' +
        '<hr class="divider">' +
        '<div class="fields">' +
        (loc ? fld('Location', loc) : '') +
        (data.grid ? fld('Grid', data.grid) : '') +
        (data.lat && data.lon ? fld('Lat / Lon', data.lat + ', ' + data.lon) : '') +
        '</div>' +
        (mapURL ? '<a class="map-link" href="' + mapURL + '" target="_blank" rel="noopener">View on map ↗</a>' : '') +
        '</div>';
    }
  } catch { res.innerHTML = '<div class="error-box">Request failed. Please try again.</div>'; }
  btn.disabled = false;
});

// ── Sites table ──────────────────────────────────────────────
async function loadSitesTable() {
  const tbody = document.getElementById('sites-tbody');
  tbody.innerHTML = '<tr><td colspan="7"><div class="spinner"></div></td></tr>';
  const sites = await fetch('/qrz/sites').then(r => r.json());
  if (!sites || !sites.length) { tbody.innerHTML = '<tr><td colspan="7" style="color:#a0aec0;padding:1rem">No sites yet.</td></tr>'; return; }
  tbody.innerHTML = sites.map(s =>
    '<tr>' +
    '<td><span class="cs-link" title="Look up and edit" onclick="lookupAndSwitch(\'' + esc(s.call_sign) + '\')">' + esc(s.call_sign) + '</span></td>' +
    '<td>' + esc(s.name) + '</td>' +
    '<td>' + s.lat.toFixed(4) + '</td>' +
    '<td>' + s.lon.toFixed(4) + '</td>' +
    '<td>' + (s.qnf != null ? s.qnf : 3) + '</td>' +
    '<td>' + (s.qnh != null ? s.qnh : '—') + '</td>' +
    '<td><div class="actions">' +
    '<button class="btn btn-primary btn-sm" onclick=\'openEditModal(' + JSON.stringify(s) + ')\'>Edit</button>' +
    '<button class="btn btn-danger btn-sm" onclick="deleteSite(\'' + esc(s.call_sign) + '\')">Delete</button>' +
    '</div></td></tr>'
  ).join('');
}

async function deleteSite(cs) {
  if (!confirm('Delete ' + cs + '?')) return;
  await fetch('/qrz/sites/' + encodeURIComponent(cs), { method: 'DELETE' });
  loadSitesTable();
}

function lookupAndSwitch(cs) {
  document.querySelectorAll('nav a').forEach(x => x.classList.remove('active'));
  document.querySelector('[data-view="lookup"]').classList.add('active');
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById('view-lookup').classList.add('active');
  document.getElementById('cs-input').value = cs;
  document.getElementById('lookup-form').dispatchEvent(new Event('submit'));
}

// ── Modal ────────────────────────────────────────────────────
let editMode = false;
let editCallsign = '';

function openAddModal() {
  editMode = false; editCallsign = '';
  document.getElementById('modal-title').textContent = 'Add Site';
  document.getElementById('f-callsign').value = '';
  document.getElementById('f-callsign').disabled = false;
  document.getElementById('f-name').value = '';
  document.getElementById('f-lat').value = '';
  document.getElementById('f-lon').value = '';
  document.getElementById('f-qnf').value = '3';
  document.getElementById('f-qnh').value = '';
  document.getElementById('modal').classList.add('open');
}

function openEditModal(s) {
  editMode = true; editCallsign = s.call_sign;
  document.getElementById('modal-title').textContent = 'Edit Site';
  document.getElementById('f-callsign').value = s.call_sign;
  document.getElementById('f-callsign').disabled = true;
  document.getElementById('f-name').value = s.name;
  document.getElementById('f-lat').value = s.lat;
  document.getElementById('f-lon').value = s.lon;
  document.getElementById('f-qnf').value = s.qnf != null ? s.qnf : 3;
  document.getElementById('f-qnh').value = s.qnh != null ? s.qnh : '';
  document.getElementById('modal').classList.add('open');
}

function closeModal() { document.getElementById('modal').classList.remove('open'); }

async function qrzFill() {
  const cs = document.getElementById('f-callsign').value.trim().toUpperCase();
  if (!cs) return;
  const data = await fetch('/qrz/lookup/' + encodeURIComponent(cs)).then(r => r.json());
  if (data.error) { alert(data.error); return; }
  document.getElementById('f-name').value = data.name || '';
  document.getElementById('f-lat').value = data.lat || '';
  document.getElementById('f-lon').value = data.lon || '';
  document.getElementById('f-qnh').value = data.altm || '';
}

document.getElementById('site-form').addEventListener('submit', async e => {
  e.preventDefault();
  const qnhVal = document.getElementById('f-qnh').value;
  const body = {
    call_sign: document.getElementById('f-callsign').value.trim().toUpperCase(),
    name: document.getElementById('f-name').value.trim(),
    lat: parseFloat(document.getElementById('f-lat').value),
    lon: parseFloat(document.getElementById('f-lon').value),
    qnf: parseFloat(document.getElementById('f-qnf').value) || 3,
    qnh: qnhVal !== '' ? parseFloat(qnhVal) : null,
  };
  const url = editMode ? '/qrz/sites/' + encodeURIComponent(editCallsign) : '/qrz/sites';
  const method = editMode ? 'PUT' : 'POST';
  const resp = await fetch(url, { method, headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
  if (resp.ok) { closeModal(); loadSitesTable(); }
  else { const d = await resp.json(); alert(d.error || 'Save failed'); }
});

// ── Map ──────────────────────────────────────────────────────
let clusterGroup = null;

async function initMap() {
  if (!leafletMap) {
    leafletMap = L.map('map').setView([51.5, -2.0], 6);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }).addTo(leafletMap);
  }
  if (clusterGroup) { leafletMap.removeLayer(clusterGroup); }
  clusterGroup = L.markerClusterGroup();

  const sites = await fetch('/qrz/sites').then(r => r.json());
  if (!sites || !sites.length) return;
  const bounds = [];
  sites.forEach(s => {
    if (!s.lat || !s.lon) return;
    const m = L.marker([s.lat, s.lon]);
    m.bindPopup('<strong>' + esc(s.call_sign) + '</strong><br>' + esc(s.name));
    clusterGroup.addLayer(m);
    bounds.push([s.lat, s.lon]);
  });
  leafletMap.addLayer(clusterGroup);
  if (bounds.length) leafletMap.fitBounds(bounds, { padding: [30, 30] });
}

// ── Helpers ──────────────────────────────────────────────────
function fld(label, value) {
  return '<div class="field"><span class="field-label">' + esc(label) + '</span><span class="field-value">' + esc(value) + '</span></div>';
}
function esc(s) {
  return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}
</script>
</body>
</html>`

func main() {
	username := os.Getenv("QRZ_USERNAME")
	password := os.Getenv("QRZ_PASSWORD")
	port := os.Getenv("QRZ_PORT")

	if username == "" || password == "" {
		log.Fatal("QRZ_USERNAME and QRZ_PASSWORD environment variables are required")
	}
	if port == "" {
		port = "8091"
	}

	t50Dir := os.Getenv("TERRAIN50_DIR")

	client := qrz.NewClient(username, password)
	db.OpenDatabase()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
	})

	// GET /sites — list all; POST /sites — create
	http.HandleFunc("/sites", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			sites, err := db.GetAllQTH()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if sites == nil {
				sites = []db.QTH{}
			}
			json.NewEncoder(w).Encode(sites)

		case http.MethodPost:
			var body struct {
				CallSign string   `json:"call_sign"`
				Name     string   `json:"name"`
				Lat      float32  `json:"lat"`
				Lon      float32  `json:"lon"`
				QNF      float64  `json:"qnf"`
				QNH      *float64 `json:"qnh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
				return
			}
			if body.QNF == 0 {
				body.QNF = 3
			}
			if err := db.AddQTH(body.CallSign, body.Name, body.Lat, body.Lon, body.QNF, body.QNH); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// PUT /sites/{callsign} — update; DELETE /sites/{callsign} — delete
	http.HandleFunc("/sites/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cs := strings.TrimPrefix(r.URL.Path, "/sites/")
		cs = strings.TrimSpace(cs)
		if cs == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}

		switch r.Method {
		case http.MethodPut:
			var body struct {
				Name string   `json:"name"`
				Lat  float32  `json:"lat"`
				Lon  float32  `json:"lon"`
				QNF  float64  `json:"qnf"`
				QNH  *float64 `json:"qnh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
				return
			}
			if body.QNF == 0 {
				body.QNF = 3
			}
			if err := db.UpdateQTH(cs, body.Name, body.Lat, body.Lon, body.QNF, body.QNH); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

		case http.MethodDelete:
			if err := db.DeleteQTH(cs); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		callsign := strings.TrimPrefix(r.URL.Path, "/lookup/")
		callsign = strings.TrimSpace(callsign)
		if callsign == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}

		result, err := client.Lookup(callsign)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if result.Lat != "" && result.Lon != "" {
			if result.AltM == "" && t50Dir != "" {
				lat, err1 := strconv.ParseFloat(result.Lat, 64)
				lon, err2 := strconv.ParseFloat(result.Lon, 64)
				if err1 == nil && err2 == nil {
					if elev, err := terrain50.ElevationAt(t50Dir, lat, lon); err == nil {
						result.AltM = strconv.FormatFloat(elev, 'f', 1, 64)
					}
				}
			}
			db.UpsertQTH(result.Callsign, result.Name, result.Lat, result.Lon, result.AltM)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Printf("qrzlook listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

