import SubmitForm from './SubmitForm';
import TaskTable from './TaskTable';

function App() {
  return (
    <div style={{ maxWidth: 700, margin: '2rem auto', padding: '0 1rem' }}>
      <h1>TaskMaster</h1>
      <SubmitForm />
      <TaskTable />
    </div>
  );
}

export default App;