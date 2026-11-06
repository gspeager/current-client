import { useState } from 'react'
import { Play, Square, X } from 'lucide-react'
import { TaskService, type TaskInfo } from '@current-client-bindings/app'
import Checkbox from '../../components/forms/Checkbox'
import { errorMessage } from '../../lib/errors'
import { useDialogs } from '../../lib/useDialogs'
import type { RunStatus, useTasks } from './useTasks'
import './TasksView.scss'

const STATUS_LABEL: Record<RunStatus, string> = {
  running: 'Running',
  passed: 'Passed',
  failed: 'Failed',
  stopped: 'Stopped',
}

function seconds(ms: number): string {
  return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(1)} s`
}

interface TasksViewProps {
  repoPath: string
  tasks: ReturnType<typeof useTasks>
}

function TasksView({ repoPath, tasks }: TasksViewProps) {
  const { prompt } = useDialogs()
  const [selected, setSelected] = useState<string | null>(null)
  const [addError, setAddError] = useState<string | null>(null)
  const list = tasks.list
  const shown: TaskInfo | undefined = list?.tasks.find((task) => task.id === selected)
  const shownRun = shown && tasks.runOf(shown.id)

  const addCommand = async () => {
    const name = await prompt({ title: 'Add a command', label: 'Name', confirmLabel: 'Next' })
    if (!name) return
    const command = await prompt({ title: 'Add a command', label: `Command for ${name}`, confirmLabel: 'Add' })
    if (!command) return
    setAddError(null)
    TaskService.AddCustom(repoPath, name, command)
      .then(tasks.reload)
      .catch((err: unknown) => setAddError(errorMessage(err)))
  }

  const removeCommand = (task: TaskInfo) => {
    TaskService.RemoveCustom(repoPath, task.name)
      .then(tasks.reload)
      .catch((err: unknown) => setAddError(errorMessage(err)))
  }

  const error = tasks.error ?? addError
  return (
    <div className="tasks-view">
      <div className="tasks-list-pane">
        {tasks.warning && (
          <p className="tasks-warning" role="status">
            {tasks.warning}
            <button type="button" aria-label="Dismiss" onClick={tasks.dismissWarning}>
              <X size={12} strokeWidth={1.75} />
            </button>
          </p>
        )}
        {error && (
          <p className="tasks-error" role="alert">
            {error}
          </p>
        )}
        {list?.tasks.length === 0 && (
          <p className="tasks-hint">No commands found in a Taskfile, Makefile, package.json or go.mod.</p>
        )}
        <ul className="tasks-list">
          {list?.tasks.map((task) => {
            const run = tasks.runOf(task.id)
            return (
              <li key={task.id} className={task.id === selected ? 'tasks-row tasks-row-selected' : 'tasks-row'}>
                {run?.status === 'running' ? (
                  <button
                    type="button"
                    className="tasks-icon"
                    aria-label={`Stop ${task.name}`}
                    title="Stop"
                    onClick={() => tasks.stop(task.id)}
                  >
                    <Square size={12} strokeWidth={1.75} />
                  </button>
                ) : (
                  <button
                    type="button"
                    className="tasks-icon"
                    aria-label={`Run ${task.name}`}
                    title="Run"
                    onClick={() => {
                      setSelected(task.id)
                      void tasks.run(task)
                    }}
                  >
                    <Play size={12} strokeWidth={1.75} />
                  </button>
                )}
                <button type="button" className="tasks-name" title={task.command} onClick={() => setSelected(task.id)}>
                  <span>{task.name}</span>
                  <span className="tasks-source">{task.source}</span>
                </button>
                {run && <span className={`tasks-status tasks-status-${run.status}`}>{STATUS_LABEL[run.status]}</span>}
                <Checkbox
                  label="Commit"
                  checked={list.beforeCommit.includes(task.id)}
                  onChange={(on) => void tasks.setChecklist('commit', task.id, on)}
                  ariaLabel={`Run ${task.name} with each commit`}
                />
                <Checkbox
                  label="Push"
                  checked={list.beforePush.includes(task.id)}
                  onChange={(on) => void tasks.setChecklist('push', task.id, on)}
                  ariaLabel={`Run ${task.name} with each push`}
                />
                {task.custom && (
                  <button
                    type="button"
                    className="tasks-icon"
                    aria-label={`Remove ${task.name}`}
                    title="Remove"
                    onClick={() => removeCommand(task)}
                  >
                    <X size={12} strokeWidth={1.75} />
                  </button>
                )}
              </li>
            )
          })}
        </ul>
        <button type="button" className="tasks-add" onClick={() => void addCommand()}>
          Add a command
        </button>
      </div>
      <div className="tasks-output-pane">
        {shown ? (
          <>
            <p className="tasks-output-header">
              <span className="tasks-output-command">{shown.command}</span>
              {shownRun && shownRun.status !== 'running' && (
                <span className="tasks-output-result">
                  {shownRun.status === 'stopped' ? 'Stopped' : `Exit ${shownRun.exitCode}`} ·{' '}
                  {seconds(shownRun.durationMs)}
                </span>
              )}
            </p>
            <pre className="tasks-output" aria-label={`Output of ${shown.name}`}>
              {shownRun?.output}
            </pre>
          </>
        ) : (
          <p className="tasks-hint">Choose a command to see its output. Output stays in memory until the app closes.</p>
        )}
      </div>
    </div>
  )
}

export default TasksView
