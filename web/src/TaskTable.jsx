import { useState, useEffect } from 'react';

const COORDINATOR_URL = 'http://localhost:8082';
const POLL_INTERVAL_MS = 2000;

// Derives a human-readable status from the task's timestamps —
// the same "status is derived, never stored directly" principle
// used throughout the backend.
function deriveStatus(task) {
  if (task.dead_letter_at) return 'dead_letter';
  if (task.needs_review_at) return 'needs_review';
  if (task.completed_at) return 'completed';
  if (task.started_at) return 'running';
  return 'pending';
}

const STATUS_STYLES = {
  completed:    { label: 'completed',    bg: '#dcfce7', color: '#16a34a' },
  running:      { label: 'running',      bg: '#dbeafe', color: '#2563eb' },
  pending:      { label: 'pending',      bg: '#f1f5f9', color: '#64748b' },
  needs_review: { label: 'needs review', bg: '#fef3c7', color: '#b45309' },
  dead_letter:  { label: 'dead letter',  bg: '#fee2e2', color: '#dc2626' },
};

function StatusBadge({ status }) {
  const style = STATUS_STYLES[status];
  return (
    <span
      style={{
        background: style.bg,
        color: style.color,
        fontSize: 12,
        padding: '2px 10px',
        borderRadius: 999,
      }}
    >
      {style.label}
    </span>
  );
}

function TaskTable() {
  const [tasks, setTasks] = useState([]);
  const [error, setError] = useState(null);

  useEffect(() => {
    async function fetchTasks() {
      try {
        const res = await fetch(`${COORDINATOR_URL}/tasks`);
        if (!res.ok) throw new Error('failed to load tasks');
        const data = await res.json();
        setTasks(data || []);
        setError(null);
      } catch (err) {
        setError('Failed to load tasks — is the coordinator running?');
      }
    }

    fetchTasks(); // fetch immediately on mount, don't wait for the first interval tick
    const interval = setInterval(fetchTasks, POLL_INTERVAL_MS);
    return () => clearInterval(interval); // cleanup on unmount
  }, []);

  async function handleRequeue(taskId) {
    try {
      const res = await fetch(`${COORDINATOR_URL}/tasks/${taskId}/requeue`, {
        method: 'POST',
      });
      if (!res.ok) {
        const text = await res.text();
        alert(`Requeue failed: ${text.trim()}`);
      }
      // No manual state update needed — the next poll picks up the change.
    } catch (err) {
      alert('Failed to reach the coordinator to requeue this task.');
    }
  }

  return (
    <div className="card">
      <p className="card-title">Tasks</p>
      {error && <p className="error-text">{error}</p>}
      <table className="task-table">
        <thead>
          <tr>
            <th>Command</th>
            <th>Worker</th>
            <th>Status</th>
            <th>Completed</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((task) => {
            const status = deriveStatus(task);
            return (
              <tr key={task.id}>
                <td className="command-cell">{task.command}</td>
                <td className="muted">{task.worker_id || '—'}</td>
                <td><StatusBadge status={status} /></td>
                <td className="muted">
                  {task.completed_at
                    ? new Date(task.completed_at).toLocaleTimeString()
                    : '—'}
                </td>
                <td>
                  {status === 'needs_review' && (
                    <button
                      className="requeue-btn"
                      onClick={() => handleRequeue(task.id)}
                    >
                      Requeue
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {tasks.length === 0 && !error && <p className="muted">No tasks yet.</p>}
    </div>
  );
}

export default TaskTable;