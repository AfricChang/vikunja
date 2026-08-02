import type {IAbstract} from './IAbstract'
import type {IUser} from './IUser'
import type {ITask} from './ITask'

export type IterationStatus = 'planned' | 'active' | 'closed'

export interface IIteration extends IAbstract {
	id: number
	title: string
	description: string
	startDate: Date
	endDate: Date
	status: IterationStatus
	previousIterationId: number
	closedAt: Date | null
	closedById: number
	created: Date
	updated: Date
	owner?: IUser
	taskCount: number
	doneTaskCount: number
	hiddenTaskCount: number
	completionRate: number
	hasNextIteration: boolean
	overlapping: boolean
}

export interface ITaskIterationHistory {
	id: number
	taskId: number
	iterationId: number
	joinedAt: Date
	joinedById: number
	leftAt: Date | null
	leftById: number
	leftReason: string
	sourceIterationId: number
	targetIterationId: number
	taskTitle: string
	projectId: number
	iterationTitle: string
	iterationStatus: IterationStatus
}

export interface ITaskIterationInfo {
	current?: IIteration
	history: ITaskIterationHistory[]
}

export interface IIterationTaskList {
	items: ITask[]
	total: number
	page: number
	perPage: number
	totalPages: number
}

export interface IIterationList {
	items: IIteration[]
	total: number
	page: number
	perPage: number
	totalPages: number
}
