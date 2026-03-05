// History page component
// Displays the user's problem-solving history using HistoryScreenComponent

import React, { useState, useEffect } from 'react';
import { getHistory } from '../api';
import HistoryScreenComponent from './HistoryScreenComponent';

const HistoryPage = ({ mockHistory }) => {
  const [history, setHistory] = useState(mockHistory || []);
  const [loading, setLoading] = useState(!mockHistory);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!mockHistory) {
      fetchHistory();
    }
  }, [mockHistory]);

  const fetchHistory = async () => {
    try {
      setLoading(true);
      const data = await getHistory();
      setHistory(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-container">

      {error && <div className="error-message">{error}</div>}

      <HistoryScreenComponent history={history} loading={loading} />

      {!loading && history.length > 0 && (
        <div style={{ textAlign: 'center', marginTop: '20px' }}>
          <button
            onClick={fetchHistory}
            className="pixel-button"
            data-testid="refresh-button"
          >
            REFRESH
          </button>
        </div>
      )}
    </div>
  );
};

export default HistoryPage;
