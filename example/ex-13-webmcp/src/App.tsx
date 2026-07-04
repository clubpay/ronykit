import { TaskBoard } from './components/TaskBoard';
import { WebMCPProvider } from './components/WebMCPProvider';
import './index.css';

export function App() {
  return (
    <WebMCPProvider>
      <main>
        <TaskBoard />
      </main>
    </WebMCPProvider>
  );
}
