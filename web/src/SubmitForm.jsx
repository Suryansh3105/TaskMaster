import { useState } from 'react';

const SCHEDULER_URL = 'http://localhost:8080';

function SubmitForm() {
  const [command, setCommand] = useState('');
  const [scheduledAt, setScheduledAt] = useState('');
  const [error, setError] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const res = await fetch(`${SCHEDULER_URL}/schedule`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          command,
          scheduled_at: new Date(scheduledAt).toISOString(),
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        setError(text.trim());
        return;
      }

      setCommand('');
      setScheduledAt('');
    } catch (err) {
      setError('Failed to reach the scheduler — is it running?');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="card">
      <p className="card-title">Submit task</p>
      <form onSubmit={handleSubmit} className="submit-form">
        <input
          type="text"
          placeholder="echo hello && sleep 2"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          required
        />
        <input
          type="datetime-local"
          value={scheduledAt}
          onChange={(e) => setScheduledAt(e.target.value)}
          required
        />
        <button type="submit" disabled={submitting}>
          {submitting ? 'Submitting...' : 'Submit task'}
        </button>
      </form>
      {error && <p className="error-text">{error}</p>}
    </div>
  );
}

export default SubmitForm;