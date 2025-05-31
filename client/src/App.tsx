import React, { useState, useEffect, ChangeEvent, FormEvent } from 'react';
import './App.css';

interface Location {
  lat: number;
  lon: number;
}

interface Sensor {
  sensor_id: number;
  name: string;
  location: Location;
  status: 'online' | 'offline';
}

interface AnalysisRecord {
  sensor_id: number;
  coeff: number;
  lag: number;
  timestamp: string;
  recording_path: string;
}

// Shape of the paginated response from /data
interface AnalysisResponse {
  totalCount: number;
  page: number;
  pageSize: number;
  records: AnalysisRecord[];
}

interface LatestRecord {
  sensor_id: number;
  coeff: number;
  lag: number;
  timestamp: string;
  recording_path: string;
}

const App: React.FC = () => {
  // --- Sensors State & Polling ---
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [sensorError, setSensorError] = useState<string | null>(null);

  // --- Analysis Records State & Pagination ---
  const [records, setRecords] = useState<AnalysisRecord[]>([]);
  const [loadingRecords, setLoadingRecords] = useState<boolean>(true);
  const [analysisError, setAnalysisError] = useState<string | null>(null);
  const [page, setPage] = useState<number>(1);
  const [pageSize] = useState<number>(10); // fixed page size of 10
  const [totalCount, setTotalCount] = useState<number>(0);

  // --- Form State for Creating a Sensor ---
  const [newSensorId, setNewSensorId] = useState<number>(0);
  const [newName, setNewName] = useState<string>('');
  const [newLat, setNewLat] = useState<number>(0);
  const [newLon, setNewLon] = useState<number>(0);
  const [newStatus, setNewStatus] = useState<'online' | 'offline'>('online');
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null);

  const [latestRecords, setLatestRecords] = useState<LatestRecord[]>([]);
  const [latestError, setLatestError] = useState<string | null>(null);

   useEffect(() => {
    fetchLatest();
    const intervalId = setInterval(fetchLatest, 2000);
    return () => clearInterval(intervalId);
  }, []);

  useEffect(() => {
    fetchSensors();
    const interval = setInterval(fetchSensors, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchSensors = () => {
    fetch('http://localhost:8080/sensors')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch sensors');
        return res.json() as Promise<Sensor[]>;
      })
      .then(data => {
        setSensors(data);
      })
      .catch(err => {
        setSensorError(err.message);
      });
  };

  // --- Fetch analysis data for current page ---
  useEffect(() => {
    fetchAnalysis(page);
  }, [page]);

  const fetchAnalysis = (requestedPage: number) => {
    setLoadingRecords(true);
    fetch(`http://localhost:8080/data?page=${requestedPage}&pageSize=${pageSize}`)
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch analysis data');
        return res.json() as Promise<AnalysisResponse>;
      })
      .then((data) => {
        setRecords(data.records);
        setTotalCount(data.totalCount);
        setLoadingRecords(false);
      })
      .catch(err => {
        setAnalysisError(err.message);
        setLoadingRecords(false);
      });
  };

  const fetchLatest = () => {
    fetch('http://localhost:8080/data/latest')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch latest records');
        return res.json() as Promise<LatestRecord[]>;
      })
      .then(data => {
        setLatestRecords(data);
      })
      .catch(err => {
        setLatestError(err.message);
      });
  };

  // --- Form Handlers ---
  const handleSensorIdChange = (e: ChangeEvent<HTMLInputElement>) => setNewSensorId(Number(e.target.value));
  const handleNameChange = (e: ChangeEvent<HTMLInputElement>) => setNewName(e.target.value);
  const handleLatChange = (e: ChangeEvent<HTMLInputElement>) => setNewLat(Number(e.target.value));
  const handleLonChange = (e: ChangeEvent<HTMLInputElement>) => setNewLon(Number(e.target.value));
  const handleStatusChange = (e: ChangeEvent<HTMLSelectElement>) => {
    setNewStatus(e.target.value as 'online' | 'offline');
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(null);

    const payload = {
      sensor_id: newSensorId,
      name: newName,
      location: { lat: newLat, lon: newLon },
      status: newStatus,
    };

    fetch('http://localhost:8080/sensors', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
      .then(res => {
        if (!res.ok) throw new Error('Failed to create sensor');
        return res.json();
      })
      .then(() => {
        setSubmitSuccess('Sensor created/updated successfully');
        setNewSensorId(0);
        setNewName('');
        setNewLat(0);
        setNewLon(0);
        setNewStatus('online');
        fetchSensors();
      })
      .catch(err => {
        setSubmitError(err.message);
      })
      .finally(() => setSubmitting(false));
  };

  // --- Pagination Controls ---
  const totalPages = Math.ceil(totalCount / pageSize);
  const handlePrevPage = () => { if (page > 1) setPage(page - 1); };
  const handleNextPage = () => { if (page < totalPages) setPage(page + 1); };

  return (
    <div className="container">
      <h1>Gateway Dashboard</h1>

      {/* Sensor Creation Form */}
      <section className="form-section">
        <h2>Create / Update Sensor</h2>
        <form onSubmit={handleSubmit} className="sensor-form">
          <div className="form-row">
            <label htmlFor="sensor-id">Sensor ID:</label>
            <input
              type="number"
              id="sensor-id"
              value={newSensorId}
              onChange={handleSensorIdChange}
              required
            />
          </div>
          <div className="form-row">
            <label htmlFor="sensor-name">Name:</label>
            <input
              type="text"
              id="sensor-name"
              value={newName}
              onChange={handleNameChange}
              required
            />
          </div>
          <div className="form-row">
            <label htmlFor="sensor-lat">Latitude:</label>
            <input
              type="number"
              step="0.000001"
              id="sensor-lat"
              value={newLat}
              onChange={handleLatChange}
              required
            />
          </div>
          <div className="form-row">
            <label htmlFor="sensor-lon">Longitude:</label>
            <input
              type="number"
              step="0.000001"
              id="sensor-lon"
              value={newLon}
              onChange={handleLonChange}
              required
            />
          </div>
          <div className="form-row">
            <label htmlFor="sensor-status">Status:</label>
            <select
              id="sensor-status"
              value={newStatus}
              onChange={handleStatusChange}
            >
              <option value="online">Online</option>
              <option value="offline">Offline</option>
            </select>
          </div>
          <button type="submit" disabled={submitting} className="submit-button">
            {submitting ? 'Submitting...' : 'Submit'}
          </button>
          {submitError && <div className="error">{submitError}</div>}
          {submitSuccess && <div className="success">{submitSuccess}</div>}
        </form>
      </section>

      <section className="table-section">
        <h2>Latest Record</h2>
        {latestError ? (
          <div className="error">{latestError}</div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Sensor ID</th>
                <th>Coefficient</th>
                <th>Lag</th>
                <th>Timestamp</th>
                <th>Recording</th>
              </tr>
            </thead>
            <tbody>
              {latestRecords.map((rec) => (
                <tr key={rec.sensor_id}>
                  <td>{rec.sensor_id}</td>
                  <td>{rec.coeff.toFixed(4)}</td>
                  <td>{rec.lag.toFixed(4)}</td>
                  <td>{new Date(rec.timestamp).toLocaleString()}</td>
		  <td>
		  {rec.recording_path ? (
			  <audio controls preload="none" style={{ width: '200px' }}>
			  <source
			  src={`http://localhost:8080/${rec.recording_path}`}
				  type="audio/wav"
			  />
			  Your browser does not support the audio element.
				  </audio>
		  ) : (
		  <em>—</em>
		  )}
		  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* Registered Sensors (polled every 10s) */}
      <section className="table-section">
        <h2>Registered Sensors</h2>
        {sensorError ? (
          <div className="error">{sensorError}</div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Sensor ID</th>
                <th>Name</th>
                <th>Latitude</th>
                <th>Longitude</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {sensors.map(sensor => (
                <tr key={sensor.sensor_id}>
                  <td>{sensor.sensor_id}</td>
                  <td>{sensor.name}</td>
                  <td>{sensor.location.lat.toFixed(6)}</td>
                  <td>{sensor.location.lon.toFixed(6)}</td>
                  <td>{sensor.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* Analysis Records (with pagination) */}
      <section className="table-section">
        <h2>Analysis Records</h2>
        {loadingRecords ? (
          <p>Loading analysis data...</p>
        ) : analysisError ? (
          <div className="error">{analysisError}</div>
        ) : (
          <>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Sensor ID</th>
                  <th>Coefficient</th>
                  <th>Lag</th>
                  <th>Timestamp</th>
                <th>Recording</th>
                </tr>
              </thead>
              <tbody>
                {records.map((rec, idx) => (
                  <tr key={idx}>
                    <td>{rec.sensor_id}</td>
                    <td>{rec.coeff.toFixed(4)}</td>
                    <td>{rec.lag.toFixed(4)}</td>
                    <td>{new Date(rec.timestamp).toLocaleString()}</td>
		    <td>
		    {rec.recording_path ? (
			    <audio controls preload="none" style={{ width: '200px' }}>
			    <source
			  src={`http://localhost:8080/${rec.recording_path}`}
				  type="audio/wav"
			  />
			    Your browser does not support the audio element.
				    </audio>
		    ) : (
		    <em>—</em>
		    )}
		    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination Controls */}
            <div className="pagination">
              <button
                onClick={handlePrevPage}
                disabled={page <= 1}
                className="page-button"
              >
                « Previous
              </button>
              <span className="page-info">
                Page {page} of {totalPages}
              </span>
              <button
                onClick={handleNextPage}
                disabled={page >= totalPages}
                className="page-button"
              >
                Next »
              </button>
            </div>
          </>
        )}
      </section>
    </div>
  );
};

export default App;
