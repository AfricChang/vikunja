<template>
	<div class="active-iteration">
		<header class="active-iteration__header">
			<div>
				<p class="active-iteration__eyebrow">
					{{ $t('iteration.home.eyebrow') }}
				</p>
				<h2 class="active-iteration__title">
					<RouterLink :to="detailRoute">
						{{ iteration.title }}
					</RouterLink>
				</h2>
				<p class="active-iteration__dates">
					{{ dateRange }} · {{ endsIn }}
				</p>
			</div>
			<XButton
				variant="secondary"
				:shadow="false"
				:to="detailRoute"
			>
				{{ $t('iteration.viewDetail') }}
			</XButton>
		</header>

		<Card
			:padding="false"
			class="has-overflow"
			:has-content="false"
			:loading="loading"
		>
			<div class="active-iteration__progress">
				<div class="active-iteration__progress-label">
					<span>{{ $t('iteration.completedTasks', {done: doneCount, total: iteration.taskCount}) }}</span>
					<strong>{{ percent }}%</strong>
				</div>
				<progress
					class="progress is-small is-success"
					:value="percent"
					max="100"
				/>
			</div>

			<ul
				v-if="tasks.length > 0"
				class="p-2 tasks"
			>
				<li
					v-for="task in tasks"
					:key="task.id"
				>
					<SingleTaskInProject
						:show-project="true"
						:the-task="task"
						:can-mark-as-done="canMarkAsDone(task)"
						@taskUpdated="updateTask"
					/>
				</li>
			</ul>
			<p
				v-else-if="!loading"
				class="active-iteration__empty"
			>
				{{ iteration.taskCount === 0 ? $t('iteration.home.noTasks') : $t('iteration.home.allDone') }}
			</p>
		</Card>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import Card from '@/components/misc/Card.vue'
import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'

import {formatDateSince, formatDisplayDate} from '@/helpers/time/formatDate'
import {useIterationService} from '@/services/iteration'
import {useProjectStore} from '@/stores/projects'
import {PERMISSIONS} from '@/constants/permissions'
import {error} from '@/message'
import type {IIteration} from '@/modelTypes/IIteration'
import type {ITask} from '@/modelTypes/ITask'

const props = defineProps<{
	iteration: IIteration
}>()

// The home page is an overview, not the iteration board — one screenful of work
// is enough, the detail view has the rest.
const TASK_LIMIT = 50

const service = useIterationService()
const projectStore = useProjectStore()

const tasks = ref<ITask[]>([])
const loading = ref(true)
// Tasks checked off since the list was loaded. The iteration's counts come from
// the server, so without this the progress bar would sit still while the user
// ticks tasks in front of it.
const doneDelta = ref(0)

const detailRoute = computed(() => ({name: 'iterations.detail', params: {id: props.iteration.id}}))

const dateRange = computed(() => `${formatDisplayDate(props.iteration.startDate)} — ${formatDisplayDate(props.iteration.endDate)}`)
const endsIn = computed(() => formatDateSince(props.iteration.endDate))

const doneCount = computed(() => Math.min(
	props.iteration.taskCount,
	Math.max(0, props.iteration.doneTaskCount + doneDelta.value),
))
const percent = computed(() => props.iteration.taskCount === 0
	? 0
	: Math.round((doneCount.value / props.iteration.taskCount) * 100))

function canMarkAsDone(task: ITask) {
	return (projectStore.projects[task.projectId]?.maxPermission ?? 0) > PERMISSIONS.READ
}

async function load(iterationID: number) {
	loading.value = true
	doneDelta.value = 0
	try {
		tasks.value = (await service.listTasks(iterationID, {
			status: 'open',
			// Same order the task list this replaces uses. Where undated tasks land
			// depends on how the database sorts NULLs, so it is not identical across
			// backends — dated tasks, the ones that matter here, are ordered either way.
			sortBy: 'due_date',
			orderBy: 'asc',
			perPage: TASK_LIMIT,
		})).items
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
}

// Mirrors ShowTasks: keep a checked-off task visible so the click can be undone,
// but move it out of the way.
function updateTask(updatedTask: ITask) {
	const index = tasks.value.findIndex(task => task.id === updatedTask.id)
	if (index === -1) {
		return
	}

	if (tasks.value[index].done !== updatedTask.done) {
		doneDelta.value += updatedTask.done ? 1 : -1
	}

	tasks.value[index] = updatedTask
	if (updatedTask.done) {
		tasks.value.splice(index, 1)
		tasks.value.push(updatedTask)
	}
}

watch(() => props.iteration.id, load, {immediate: true})
</script>

<style lang="scss" scoped>
.active-iteration__header {
	display: flex;
	justify-content: space-between;
	align-items: end;
	gap: 1rem;
	margin-block-end: .75rem;
}

.active-iteration__eyebrow {
	text-transform: uppercase;
	letter-spacing: .1em;
	color: var(--grey-500);
	font-size: .75rem;
	margin: 0;
}

.active-iteration__title {
	font-size: 1.5rem;
	margin: 0;
}

.active-iteration__dates {
	color: var(--grey-500);
	font-size: .8rem;
	margin: 0;
}

.active-iteration__progress {
	padding: .875rem .875rem 0;
}

.active-iteration__progress-label {
	display: flex;
	justify-content: space-between;
	gap: .75rem;
	font-size: .85rem;
	margin-block-end: .35rem;
}

.active-iteration__empty {
	text-align: center;
	color: var(--grey-500);
	padding: 2rem 1rem;
	margin: 0;
}

.tasks {
	list-style: none;
	margin: 0;
}

@media screen and (max-width: $tablet) {
	.active-iteration__header {
		align-items: stretch;
		flex-direction: column;
	}
}
</style>
