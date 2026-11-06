import { useCallback, useEffect, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import { TaskService, type TaskInfo } from '@current-client-bindings/app'
import { errorMessage } from '../../lib/errors'
import { useAsyncData } from '../../lib/useAsyncData'
import { useDialogs } from '../../lib/useDialogs'

const OUTPUT_EVENT = 'task:output'
const DONE_EVENT = 'task:done'
// What's kept of a run's output, from the end, in memory only.
const MOST_OUTPUT = 2 << 20
let nextRun = 0

export type RunStatus = 'running' | 'passed' | 'failed' | 'stopped'

export interface TaskRun {
  runId: string
  status: RunStatus
  output: string
  exitCode: number
  durationMs: number
}

// Colors and cursor moves, which a captured run has no use for.
const ESCAPES = /\x1b(?:\[[0-9;?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/g // eslint-disable-line no-control-regex

export function useTasks(repoPath: string | null, refreshKey: unknown) {
  const { confirm } = useDialogs()
  const {
    data: list,
    error: listError,
    reload,
  } = useAsyncData(() => (repoPath ? TaskService.List(repoPath) : null), [repoPath], { refreshKey })
  const [runs, setRuns] = useState<Record<string, TaskRun>>({})
  const [error, setError] = useState<string | null>(null)
  const [warning, setWarning] = useState<{ repo: string; text: string } | null>(null)
  const taskOfRun = useRef(new Map<string, string>())
  // Read when starting one, as a closure kept from before a push can be behind `runs`.
  const running = useRef(new Set<string>())
  const decoders = useRef(new Map<string, TextDecoder>())
  const checklistRuns = useRef(new Map<string, string>())

  useEffect(() => {
    const offOutput = Events.On(OUTPUT_EVENT, ({ data }: { data: { runId: string; data: string } }) => {
      const taskKey = taskOfRun.current.get(data.runId)
      if (!taskKey) return
      let decoder = decoders.current.get(data.runId)
      if (!decoder) {
        decoder = new TextDecoder()
        decoders.current.set(data.runId, decoder)
      }
      const bytes = Uint8Array.from(atob(data.data), (char) => char.charCodeAt(0))
      const text = decoder.decode(bytes, { stream: true }).replace(ESCAPES, '').replace(/\r\n/g, '\n')
      setRuns((prev) => {
        const run = prev[taskKey]
        if (!run || run.runId !== data.runId) return prev
        const output = (run.output + text).slice(-MOST_OUTPUT)
        return { ...prev, [taskKey]: { ...run, output } }
      })
    })
    const offDone = Events.On(
      DONE_EVENT,
      ({ data }: { data: { runId: string; exitCode: number; durationMs: number; stopped: boolean } }) => {
        const taskKey = taskOfRun.current.get(data.runId)
        if (!taskKey) return
        decoders.current.delete(data.runId)
        taskOfRun.current.delete(data.runId)
        running.current.delete(taskKey)
        const status: RunStatus = data.stopped ? 'stopped' : data.exitCode === 0 ? 'passed' : 'failed'
        setRuns((prev) => {
          const run = prev[taskKey]
          if (!run || run.runId !== data.runId) return prev
          return { ...prev, [taskKey]: { ...run, status, exitCode: data.exitCode, durationMs: data.durationMs } }
        })
        const checklist = checklistRuns.current.get(data.runId)
        if (checklist && status === 'failed') {
          setWarning({
            repo: taskKey.split('\0')[0],
            text: `${checklist} failed after the last commit or push. It didn't stop it.`,
          })
        }
        checklistRuns.current.delete(data.runId)
      },
    )
    return () => {
      offOutput()
      offDone()
    }
  }, [])

  const key = useCallback((id: string) => `${repoPath}\0${id}`, [repoPath])

  // The run is known before it starts, since a quick one can finish before the
  // call that started it returns.
  const launch = async (task: TaskInfo, checklist = false) => {
    if (!repoPath) return
    const taskKey = key(task.id)
    if (running.current.has(taskKey)) return
    running.current.add(taskKey)
    setError(null)
    nextRun += 1
    const runId = `run-${Date.now()}-${nextRun}`
    taskOfRun.current.set(runId, taskKey)
    if (checklist) checklistRuns.current.set(runId, task.name)
    setRuns((prev) => ({ ...prev, [taskKey]: { runId, status: 'running', output: '', exitCode: 0, durationMs: 0 } }))
    try {
      await TaskService.Run(repoPath, task.id, runId)
    } catch (err: unknown) {
      setError(errorMessage(err))
      taskOfRun.current.delete(runId)
      checklistRuns.current.delete(runId)
      running.current.delete(taskKey)
      setRuns((prev) => {
        const next = { ...prev }
        delete next[taskKey]
        return next
      })
    }
  }

  // Shows the command and folder before its first run, and again if it has changed.
  // The list is read again first, since editing a Makefile moves no ref.
  const run = async (listed: TaskInfo) => {
    if (!repoPath) return
    const fresh = await TaskService.List(repoPath).catch(() => null)
    const task = fresh?.tasks.find((item) => item.id === listed.id) ?? listed
    if (!task.approved) {
      const confirmed = await confirm({
        title: 'Run this command?',
        message: [
          task.command,
          `In ${repoPath}.`,
          task.file ? `From ${task.file}, which hasn't been approved as it is now.` : 'Not approved yet.',
          task.note,
        ]
          .filter(Boolean)
          .join('\n'),
        confirmLabel: 'Run',
      })
      if (!confirmed) return
      try {
        await TaskService.Approve(repoPath, task.id, task.fingerprint)
      } catch (err: unknown) {
        setError(errorMessage(err))
        return
      }
      reload()
    }
    await launch(task)
  }

  const stop = (taskId: string) => {
    const current = runs[key(taskId)]
    if (current) void TaskService.Stop(current.runId).catch(() => undefined)
  }

  // Alongside a commit or push, never in its way. Anything not yet approved is
  // skipped and named, since nothing runs without having been seen.
  const runChecklist = async (kind: 'commit' | 'push') => {
    if (!repoPath) return
    const fresh = await TaskService.List(repoPath).catch(() => null)
    if (!fresh) return
    const ids = kind === 'commit' ? fresh.beforeCommit : fresh.beforePush
    const chosen = fresh.tasks.filter((task) => ids.includes(task.id))
    const skipped = chosen.filter((task) => !task.approved)
    setWarning(
      skipped.length > 0
        ? { repo: repoPath, text: `Not run, since not approved: ${skipped.map((t) => t.name).join(', ')}.` }
        : null,
    )
    for (const task of chosen) {
      if (task.approved) void launch(task, true)
    }
  }

  const setChecklist = async (kind: 'commit' | 'push', id: string, on: boolean) => {
    if (!repoPath || !list) return
    const current = kind === 'commit' ? list.beforeCommit : list.beforePush
    const next = on ? [...current, id] : current.filter((item) => item !== id)
    await TaskService.SetChecklist(repoPath, kind, next).catch((err: unknown) => setError(errorMessage(err)))
    reload()
  }

  const runOf = (taskId: string): TaskRun | undefined => runs[key(taskId)]

  return {
    list,
    error: error ?? listError,
    warning: warning?.repo === repoPath ? warning.text : null,
    dismissWarning: () => setWarning(null),
    run,
    stop,
    runOf,
    runChecklist,
    setChecklist,
    reload,
  }
}
