package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/quanglewangle/qrzlook/db"
	"github.com/quanglewangle/qrzlook/qrz"
)

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>QRZ Callsign Lookup</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: #f0f4f8;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 2rem 1rem;
    color: #1a202c;
  }

  h1 {
    font-size: 1.8rem;
    font-weight: 700;
    margin-bottom: 0.25rem;
    color: #2d3748;
  }

  .subtitle {
    color: #718096;
    margin-bottom: 2rem;
    font-size: 0.95rem;
  }

  form {
    display: flex;
    gap: 0.5rem;
    width: 100%;
    max-width: 480px;
    margin-bottom: 2rem;
  }

  input[type="text"] {
    flex: 1;
    padding: 0.75rem 1rem;
    font-size: 1.1rem;
    border: 2px solid #cbd5e0;
    border-radius: 8px;
    outline: none;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    transition: border-color 0.2s;
  }

  input[type="text"]:focus { border-color: #4299e1; }

  button {
    padding: 0.75rem 1.5rem;
    font-size: 1rem;
    font-weight: 600;
    background: #3182ce;
    color: white;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    transition: background 0.2s;
    white-space: nowrap;
  }

  button:hover { background: #2b6cb0; }
  button:disabled { background: #a0aec0; cursor: not-allowed; }

  #result {
    width: 100%;
    max-width: 480px;
  }

  .card {
    background: white;
    border-radius: 12px;
    padding: 1.5rem;
    box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  }

  .callsign {
    font-size: 2rem;
    font-weight: 800;
    color: #2b6cb0;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
  }

  .name {
    font-size: 1.2rem;
    color: #2d3748;
    margin-bottom: 1.25rem;
    font-weight: 500;
  }

  .divider {
    border: none;
    border-top: 1px solid #e2e8f0;
    margin-bottom: 1.25rem;
  }

  .fields { display: flex; flex-direction: column; gap: 0.6rem; }

  .field {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 1rem;
  }

  .field-label {
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #a0aec0;
    flex-shrink: 0;
  }

  .field-value {
    font-size: 0.95rem;
    color: #2d3748;
    text-align: right;
  }

  .map-link {
    display: inline-block;
    margin-top: 1.25rem;
    font-size: 0.85rem;
    color: #3182ce;
    text-decoration: none;
  }
  .map-link:hover { text-decoration: underline; }

  .error {
    background: #fff5f5;
    border: 1px solid #fed7d7;
    color: #c53030;
    padding: 1rem 1.25rem;
    border-radius: 8px;
    font-size: 0.95rem;
  }

  .spinner {
    width: 32px; height: 32px;
    border: 3px solid #e2e8f0;
    border-top-color: #3182ce;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    margin: 1rem auto;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<h1>QRZ Lookup</h1>
<p class="subtitle">Amateur radio callsign lookup</p>

<form id="form">
  <input type="text" id="callsign" placeholder="e.g. G8GDS" autocomplete="off" autocorrect="off" spellcheck="false">
  <button type="submit" id="btn">Look up</button>
</form>

<div id="result"></div>

<script>
const form = document.getElementById('form');
const input = document.getElementById('callsign');
const btn = document.getElementById('btn');
const result = document.getElementById('result');

form.addEventListener('submit', async e => {
  e.preventDefault();
  const cs = input.value.trim().toUpperCase();
  if (!cs) return;

  btn.disabled = true;
  result.innerHTML = '<div class="spinner"></div>';

  try {
    const resp = await fetch('/qrz/lookup/' + encodeURIComponent(cs));
    const data = await resp.json();

    if (data.error) {
      result.innerHTML = '<div class="error">' + escHtml(data.error) + '</div>';
    } else {
      const location = [data.city, data.state, data.country].filter(Boolean).join(', ');
      const mapURL = data.lat && data.lon
        ? 'https://www.openstreetmap.org/?mlat=' + data.lat + '&mlon=' + data.lon + '&zoom=10'
        : null;

      result.innerHTML = '<div class="card">' +
        '<div class="callsign">' + escHtml(data.callsign) + '</div>' +
        '<div class="name">' + escHtml(data.name || '—') + '</div>' +
        '<hr class="divider">' +
        '<div class="fields">' +
        (location ? field('Location', location) : '') +
        (data.grid ? field('Grid', data.grid) : '') +
        (data.lat && data.lon ? field('Lat / Lon', data.lat + ', ' + data.lon) : '') +
        '</div>' +
        (mapURL ? '<a class="map-link" href="' + mapURL + '" target="_blank" rel="noopener">View on map ↗</a>' : '') +
        '</div>';
    }
  } catch (err) {
    result.innerHTML = '<div class="error">Request failed. Please try again.</div>';
  } finally {
    btn.disabled = false;
  }
});

function field(label, value) {
  return '<div class="field"><span class="field-label">' + escHtml(label) +
    '</span><span class="field-value">' + escHtml(value) + '</span></div>';
}

function escHtml(s) {
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

	db.OpenDatabase()
	client := qrz.NewClient(username, password)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
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
			db.UpsertQTH(result.Callsign, result.Lat, result.Lon)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Printf("qrzlook listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
