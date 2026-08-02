import {AuthenticatedHTTPFactory, apiV2Url} from '@/helpers/fetcher'
import {objectToCamelCase, objectToSnakeCase} from '@/helpers/case'
import TaskModel from '@/models/task'

import type {ITask} from '@/modelTypes/ITask'
import type {
	IIteration,
	IIterationList,
	IIterationTaskList,
	ITaskIterationHistory,
	ITaskIterationInfo,
} from '@/modelTypes/IIteration'

function parseIteration(raw: Record<string, unknown>): IIteration {
	const value = objectToCamelCase(raw)
	return {
		...value,
		id: Number(value.id),
		previousIterationId: Number(value.previousIterationId ?? 0),
		closedById: Number(value.closedById ?? 0),
		startDate: new Date(value.startDate),
		endDate: new Date(value.endDate),
		closedAt: value.closedAt ? new Date(value.closedAt) : null,
		created: new Date(value.created),
		updated: new Date(value.updated),
		taskCount: Number(value.taskCount ?? 0),
		doneTaskCount: Number(value.doneTaskCount ?? 0),
		hiddenTaskCount: Number(value.hiddenTaskCount ?? 0),
		completionRate: Number(value.completionRate ?? 0),
	} as IIteration
}

function parseHistory(raw: Record<string, unknown>): ITaskIterationHistory {
	const value = objectToCamelCase(raw)
	return {
		...value,
		joinedAt: new Date(value.joinedAt),
		leftAt: value.leftAt ? new Date(value.leftAt) : null,
	} as ITaskIterationHistory
}

function pageParams(params: Record<string, unknown>) {
	return objectToSnakeCase(params)
}

export function useIterationService() {
	const http = AuthenticatedHTTPFactory()

	async function list(params: Record<string, unknown> = {}): Promise<IIterationList> {
		const {data} = await http.get(apiV2Url('iterations'), {params: pageParams(params)})
		return {
			items: (data.items ?? []).map(parseIteration),
			total: data.total,
			page: data.page,
			perPage: data.per_page,
			totalPages: data.total_pages,
		}
	}

	async function get(id: number): Promise<IIteration> {
		const {data} = await http.get(apiV2Url(`iterations/${id}`))
		return parseIteration(data)
	}

	async function create(iteration: Partial<IIteration>): Promise<IIteration> {
		const {data} = await http.post(apiV2Url('iterations'), objectToSnakeCase(iteration))
		return parseIteration(data)
	}

	async function update(iteration: Partial<IIteration> & {id: number}): Promise<IIteration> {
		const {data} = await http.put(apiV2Url(`iterations/${iteration.id}`), objectToSnakeCase(iteration))
		return parseIteration(data)
	}

	async function remove(id: number): Promise<void> {
		await http.delete(apiV2Url(`iterations/${id}`))
	}

	async function listTasks(id: number, params: Record<string, unknown> = {}): Promise<IIterationTaskList> {
		const {data} = await http.get(apiV2Url(`iterations/${id}/tasks`), {params: pageParams(params)})
		return {
			items: (data.items ?? []).map((task: Partial<ITask>) => new TaskModel(task)),
			total: data.total,
			page: data.page,
			perPage: data.per_page,
			totalPages: data.total_pages,
		}
	}

	async function searchTasks(params: Record<string, unknown> = {}): Promise<IIterationTaskList> {
		const {data} = await http.get(apiV2Url('tasks'), {params: pageParams(params)})
		return {
			items: (data.items ?? []).map((task: Partial<ITask>) => new TaskModel(task)),
			total: data.total,
			page: data.page,
			perPage: data.per_page,
			totalPages: data.total_pages,
		}
	}

	async function addTasks(id: number, taskIds: number[], confirmTransfer = false): Promise<IIteration> {
		const {data} = await http.post(apiV2Url(`iterations/${id}/tasks/bulk`), {
			task_ids: taskIds,
			confirm_transfer: confirmTransfer,
		})
		return parseIteration(data)
	}

	async function removeTasks(id: number, taskIds: number[]): Promise<void> {
		await http.delete(apiV2Url(`iterations/${id}/tasks/bulk`), {data: {task_ids: taskIds}})
	}

	async function moveTasks(id: number, targetIterationId: number, taskIds: number[]): Promise<IIteration> {
		const {data} = await http.post(apiV2Url(`iterations/${id}/tasks/move`), {
			task_ids: taskIds,
			target_iteration_id: targetIterationId,
		})
		return parseIteration(data)
	}

	async function start(id: number): Promise<IIteration> {
		const {data} = await http.post(apiV2Url(`iterations/${id}/start`))
		return parseIteration(data)
	}

	async function reopen(id: number): Promise<IIteration> {
		const {data} = await http.post(apiV2Url(`iterations/${id}/reopen`))
		return parseIteration(data)
	}

	async function createNext(id: number, iteration: Partial<IIteration>): Promise<IIteration> {
		const {data} = await http.post(apiV2Url(`iterations/${id}/next`), objectToSnakeCase(iteration))
		return parseIteration(data)
	}

	async function close(id: number, targetIterationId: number, taskIds: number[]) {
		const {data} = await http.post(apiV2Url(`iterations/${id}/close`), {
			target_iteration_id: targetIterationId,
			rollover_task_ids: taskIds,
		})
		const value = objectToCamelCase(data)
		return {
			...value,
			iteration: parseIteration(data.iteration),
			targetIteration: data.target_iteration ? parseIteration(data.target_iteration) : null,
		}
	}

	async function getTaskIterations(taskId: number): Promise<ITaskIterationInfo> {
		const {data} = await http.get(apiV2Url(`tasks/${taskId}/iterations`))
		return {
			current: data.current ? parseIteration(data.current) : undefined,
			history: (data.history ?? []).map(parseHistory),
		}
	}

	async function setTaskIteration(taskId: number, iterationId: number, confirmTransfer = false): Promise<ITaskIterationInfo> {
		const {data} = await http.put(apiV2Url(`tasks/${taskId}/iteration`), {
			iteration_id: iterationId,
			confirm_transfer: confirmTransfer,
		})
		return {
			current: data.current ? parseIteration(data.current) : undefined,
			history: (data.history ?? []).map(parseHistory),
		}
	}

	return {list, get, create, update, remove, listTasks, searchTasks, addTasks, removeTasks, moveTasks, start, reopen, createNext, close, getTaskIterations, setTaskIteration}
}
