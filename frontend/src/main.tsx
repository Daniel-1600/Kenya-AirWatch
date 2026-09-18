import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import 'leaflet/dist/leaflet.css';
import './styles.css';

type Obs = { id: number; observed_at: string; value: number; unit: string; source: string };
type Anomaly = { id: number; observation_id: number; observed_at: string; value: number; baseline: number; deviation_percent: number; z_score: number; severity: string };
type Site = { id: number; name: string; latitude: number; longitude: number; radius_km: number };
type Summary = { baseline: number; difference_percent: number; status: string };

const API = (import.meta.env.VITE_API_URL || '/api').replace(/\/$/, '');
const fmt = (date: string) => new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short' }).format(new Date(date));
const getJSON = async <T,>(url: string, signal: AbortSignal): Promise<T> => {
  const response = await fetch(url, { signal });
  if (!response.ok) throw new Error(`Request failed (${response.status})`);
  return response.json();
};

function MapCard({ site }: { site: Site }) {
  const el = React.useRef<HTMLDivElement>(null);
  useEffect(() => {
    let map: any;
    let cancelled = false;
    (async () => {
      const L = await import('leaflet');
      if (!el.current || cancelled) return;
      map = L.map(el.current, { zoomControl: false }).setView([site.latitude, site.longitude], 13);
      L.control.zoom({ position: 'bottomright' }).addTo(map);
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', { attribution: '© OpenStreetMap contributors' }).addTo(map);
      L.circle([site.latitude, site.longitude], { radius: site.radius_km * 1000, color: '#ff7b4d', weight: 1.5, fillColor: '#ff7b4d', fillOpacity: .08 }).addTo(map);
      L.marker([site.latitude, site.longitude], { icon: L.divIcon({ className: 'site-pin', html: '<span>✦</span>', iconSize: [34, 34], iconAnchor: [17, 17] }) }).addTo(map).bindPopup(site.name);
    })();
    return () => { cancelled = true; map?.remove(); };
  }, [site]);

  return <div className="map-card">
    <div ref={el} className="map" />
    <div className="map-overlay">
      <span className="eyebrow">ANALYSIS REGION</span>
      <strong>{site.radius_km.toFixed(1)} km radius around the site</strong>
      <small>The ring is the satellite averaging area. Point-level methane sources are not inferred.</small>
    </div>
    <div className="map-key"><span><i className="site-key" /> Site location</span><span><i className="region-key" /> Analysis boundary</span></div>
  </div>;
}

function App() {
  const [sites, setSites] = useState<Site[]>([]);
  const [siteId, setSiteId] = useState(1);
  const [site, setSite] = useState<Site>({ id: 1, name: 'Dandora Dumpsite', latitude: -1.2467, longitude: 36.9068, radius_km: 4.5 });
  const [obs, setObs] = useState<Obs[]>([]);
  const [alerts, setAlerts] = useState<Anomaly[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [range, setRange] = useState(90);
  const [selected, setSelected] = useState<Anomaly | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    getJSON<Site[]>(`${API}/sites`, controller.signal).then(value => {
      setSites(value || []);
      if (value?.length && !value.some(item => item.id === siteId)) setSiteId(value[0].id);
    }).catch(err => { if (err.name !== 'AbortError') setError('The monitoring service is unavailable.'); });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    Promise.all([
      getJSON<Site>(`${API}/sites/${siteId}`, controller.signal),
      getJSON<Obs[]>(`${API}/sites/${siteId}/observations?pollutant=CH4`, controller.signal),
      getJSON<Anomaly[]>(`${API}/sites/${siteId}/anomalies`, controller.signal),
      getJSON<Summary>(`${API}/sites/${siteId}/summary?pollutant=CH4`, controller.signal).catch(() => null),
    ]).then(([nextSite, observations, anomalies, nextSummary]) => {
      setSite(nextSite);
      setObs(observations || []);
      setAlerts(anomalies || []);
      setSummary(nextSummary);
      setSelected(null);
    }).catch(err => {
      if (err.name !== 'AbortError') {
        setObs([]);
        setAlerts([]);
        setSummary(null);
        setError('The monitoring data could not be loaded. Try again when the API is available.');
      }
    }).finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [siteId]);

  const cutoff = useMemo(() => {
    const newest = obs.at(-1);
    if (!newest) return 0;
    return new Date(newest.observed_at).getTime() - range * 24 * 60 * 60 * 1000;
  }, [obs, range]);
  const shown = useMemo(() => obs.filter(item => new Date(item.observed_at).getTime() >= cutoff), [obs, cutoff]);
  const latest = obs.at(-1);
  const shownAlerts = useMemo(() => alerts.filter(item => new Date(item.observed_at).getTime() >= cutoff), [alerts, cutoff]);
  const chart = shown.map(item => ({ date: fmt(item.observed_at), value: item.value }));
  const dataMode = latest?.source.toUpperCase().includes('FIXTURE') ? 'DEMO DATA' : latest ? 'SATELLITE DATA' : 'NO DATA';
  const status = summary?.status?.toUpperCase() || 'NO DATA';
  const diff = summary?.difference_percent;
  const signed = diff == null ? '—' : `${diff >= 0 ? '+' : ''}${diff.toFixed(1)}%`;

  return <main>
    <header>
      <div className="brand"><div className="brand-mark">✦</div><div><h1>Kenya <span>AirWatch</span></h1><p>Satellite environmental monitoring</p></div></div>
      <div className="site-select"><span className="eyebrow">MONITORING SITE</span><select value={siteId} onChange={event => setSiteId(Number(event.target.value))} disabled={!sites.length}>{(sites.length ? sites : [site]).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select><small>Kenya <b>·</b> Sentinel-5P / TROPOMI</small></div>
      <div className={`live ${dataMode === 'DEMO DATA' ? 'demo' : ''}`}><i /> {loading ? 'LOADING' : error ? 'API OFFLINE' : dataMode}</div>
    </header>
    {error && <div className="error-banner" role="alert">{error}</div>}
    <section className="hero"><div><span className="eyebrow">ATMOSPHERIC METHANE OBSERVATIONS</span><h2>A clearer signal for a complex airscape.</h2><p>Track how methane concentrations around {site.name} change over time, with simple statistical alerts for unusual readings.</p></div><div className="hero-note"><span>DATA NOTE</span><strong>Atmospheric concentrations around the site</strong><small>Satellite observations cannot attribute an anomaly to the landfill alone.</small></div></section>
    <section className="stats">
      <div className={`status stat ${status.toLowerCase().replace(' ', '-')}`}><span className="eyebrow">CURRENT STATUS</span><div className="status-row"><strong>{loading ? 'LOADING' : status}</strong><i>{summary && summary.difference_percent >= 0 ? '↗' : summary ? '↘' : '—'}</i></div><small>{summary ? 'Backend z-score classification' : 'Waiting for valid observations'}</small></div>
      <div className="stat"><span className="eyebrow">LATEST METHANE</span><strong>{latest ? latest.value.toFixed(1) : '—'} {latest && <em>{latest.unit}</em>}</strong><small>{latest ? 'Sentinel-5P column mixing ratio' : 'No observation available'}</small></div>
      <div className="stat"><span className="eyebrow">CHANGE FROM BASELINE</span><strong className="accent">{signed}</strong><small>{summary ? `Reference average · ${summary.baseline.toFixed(1)} ppb` : 'No baseline available'}</small></div>
      <div className="stat"><span className="eyebrow">LAST OBSERVATION</span><strong>{latest ? fmt(latest.observed_at) : '—'}</strong><small>{latest ? `${new Date(latest.observed_at).getFullYear()} · satellite overpass` : 'No observation available'}</small></div>
    </section>
    <section className="grid"><div><div className="section-head"><div><span className="eyebrow">GEOGRAPHIC VIEW</span><h3>Analysis region</h3></div><span className="pill">CH₄ · ppb</span></div><MapCard site={site} /></div><div><div className="section-head"><div><span className="eyebrow">HISTORICAL SIGNAL</span><h3>Methane over time</h3></div><div className="range">{[30, 60, 90].map(days => <button className={range === days ? 'active' : ''} onClick={() => setRange(days)} key={days}>{days}D</button>)}</div></div><div className="chart-card"><div className="chart-kpi"><strong>{shown.length}</strong><span>valid observations<br />in the last {range} days of data</span></div>{chart.length ? <ResponsiveContainer width="100%" height={250}><AreaChart data={chart} margin={{ top: 10, right: 4, left: -20, bottom: 0 }}><defs><linearGradient id="fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#c6e86b" stopOpacity={.34} /><stop offset="100%" stopColor="#c6e86b" stopOpacity={0} /></linearGradient></defs><XAxis dataKey="date" tickLine={false} axisLine={false} tick={{ fill: '#738078', fontSize: 11 }} interval="preserveStartEnd" /><YAxis domain={['dataMin - 20', 'dataMax + 20']} tickLine={false} axisLine={false} tick={{ fill: '#738078', fontSize: 11 }} /><Tooltip contentStyle={{ background: '#13241f', border: 'none', borderRadius: 10, color: '#fff' }} formatter={(value: any) => [`${Number(value).toFixed(1)} ppb`, 'Methane']} /><Area type="monotone" dataKey="value" stroke="#b7df58" strokeWidth={2.5} fill="url(#fill)" connectNulls dot={{ r: 2, fill: '#b7df58', strokeWidth: 0 }} /></AreaChart></ResponsiveContainer> : <div className="empty-state">No valid methane observations are available for this period.</div>}<div className="chart-foot"><span><i className="line" /> methane observations</span><span>Source: {latest?.source || 'not available'}</span></div></div></div></section>
    <section className="alerts"><div className="section-head"><div><span className="eyebrow">AUTOMATED REVIEW</span><h3>Environmental alerts <span>{shownAlerts.length}</span></h3></div><small className="method">Mean · standard deviation · z-score</small></div><div className="alert-list">{shownAlerts.length ? shownAlerts.slice(0, 4).map(item => <button className="alert" key={item.id} onClick={() => setSelected(item)}><div className="alert-date"><strong>{fmt(item.observed_at)}</strong><small>{new Date(item.observed_at).getFullYear()}</small></div><span className={`severity ${item.severity}`}>{item.severity}</span><div className="alert-value"><strong>{item.value.toFixed(1)} <small>ppb</small></strong><span>baseline {item.baseline.toFixed(1)} · {item.deviation_percent >= 0 ? '+' : ''}{item.deviation_percent.toFixed(1)}%</span></div><span className="arrow">→</span></button>) : <div className="empty-alerts">No statistical alerts were detected in the selected period.</div>}</div></section>
    {selected && <div className="modal-backdrop" onClick={() => setSelected(null)}><div className="modal" onClick={event => event.stopPropagation()}><button onClick={() => setSelected(null)} className="close">×</button><span className="eyebrow">ALERT DETAIL</span><h3>{fmt(selected.observed_at)} observation</h3><div className="modal-grid"><div><small>METHANE</small><strong>{selected.value.toFixed(1)} ppb</strong></div><div><small>BASELINE</small><strong>{selected.baseline.toFixed(1)} ppb</strong></div><div><small>DEVIATION</small><strong>{selected.deviation_percent >= 0 ? '+' : ''}{selected.deviation_percent.toFixed(1)}%</strong></div><div><small>Z-SCORE</small><strong>{selected.z_score.toFixed(2)}</strong></div></div><p>Unusual atmospheric methane observation detected near {site.name}. This is an early-warning signal, not an attribution of landfill emissions.</p></div></div>}
  </main>;
}

createRoot(document.getElementById('root')!).render(<App />);
