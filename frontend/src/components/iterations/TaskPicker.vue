<template>
	<div class="task-picker">
		<div class="field">
			<div class="control has-icons-left">
				<input
					v-model="query"
					class="input"
					:placeholder="$t('iteration.tasks.search')"
					@input="search"
				>
				<span class="icon is-left"><Icon icon="search" /></span>
			</div>
		</div>
		<div class="task-picker__filters">
			<div>
				<p class="label">
					{{ $t('iteration.tasks.projects') }}
				</p>
				<label
					v-for="project in projects"
					:key="project.id"
					class="checkbox task-picker__project"
				>
					<input
						v-model="selectedProjects"
						type="checkbox"
						:value="project.id"
						@change="search"
					>
					{{ project.title }}
				</label>
				<label class="checkbox task-picker__project">
					<input
						v-model="hideDone"
						type="checkbox"
						@change="search"
					>
					{{ $t('iteration.tasks.hideDone') }}
				</label>
			</div>
			<div
				class="task-picker__results loader-container"
				:class="{'is-loading': loading}"
			>
				<label
					v-for="task in tasks"
					:key="task.id"
					class="task-picker__row"
				>
					<input
						v-model="selected"
						type="checkbox"
						:value="task.id"
					>
					<span>
						<strong>{{ task.title }}</strong>
						<small>{{ task.identifier || `#${task.id}` }}</small>
					</span>
					<span>{{ projectTitle(task.projectId) }}</span>
					<span>{{ task.done ? $t('task.attributes.done') : $t('iteration.tasks.open') }}</span>
				</label>
				<p
					v-if="!loading && tasks.length === 0"
					class="has-text-grey has-text-centered p-4"
				>
					{{ $t('iteration.tasks.emptyPicker') }}
				</p>
			</div>
		</div>
		<p class="help">
			{{ $t('iteration.tasks.permissionHint') }}
		</p>
		<div class="task-picker__actions">
			<XButton
				variant="tertiary"
				@click="$emit('cancel')"
			>
				{{ $t('misc.cancel') }}
			</XButton>
			<XButton
				:disabled="selected.length === 0"
				:loading="saving"
				@click="$emit('add', selected)"
			>
				{{ $t('iteration.tasks.addSelected', {count: selected.length}) }}
			</XButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue'

import {useIterationService} from '@/services/iteration'
import {useProjectStore} from '@/stores/projects'
import type {ITask} from '@/modelTypes/ITask'

defineProps<{saving?: boolean}>()
defineEmits<{add: [taskIds: number[]], cancel: []}>()

const service = useIterationService()
const projectStore = useProjectStore()
const projects = computed(() => projectStore.projectsArray.filter(project => project.id > 0 && !project.isArchived))
const selectedProjects = ref<number[]>([])
const hideDone = ref(true)
const query = ref('')
const tasks = ref<ITask[]>([])
const selected = ref<number[]>([])
const loading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

function projectTitle(id: number) {
	return projectStore.projects[id]?.title ?? `#${id}`
}

async function runSearch() {
	loading.value = true
	try {
		const filters: string[] = []
		if (hideDone.value) filters.push('done = false')
		if (selectedProjects.value.length) filters.push(`project_id in ${selectedProjects.value.join(', ')}`)
		const result = await service.searchTasks({
			q: query.value,
			filter: filters.join(' && '),
			sortBy: ['done', 'id'],
			orderBy: ['asc', 'desc'],
			perPage: 100,
		})
		tasks.value = result.items
	} finally {
		loading.value = false
	}
}

function search() {
	if (searchTimer) clearTimeout(searchTimer)
	searchTimer = setTimeout(runSearch, 250)
}

onMounted(runSearch)
</script>

<style scoped lang="scss">
.task-picker__filters {
	display: grid;
	grid-template-columns: minmax(12rem, 14rem) minmax(0, 1fr);
	gap: 1.25rem;
}

.task-picker__filters > :first-child,
.task-picker__filters > :first-child .label {
	text-align: start;
}

.task-picker__project {
	display: flex;
	inline-size: 100%;
	padding-block: .4rem;
	text-align: start;
}
.task-picker__results {
	max-block-size: 24rem;
	overflow: auto;
	border: 1px solid var(--card-border-color);
	border-radius: $radius;
	text-align: start;
}

.task-picker__row {
	display: grid;
	grid-template-columns: 1.5rem minmax(12rem, 1fr) minmax(8rem, .5fr) 6rem;
	gap: .75rem;
	align-items: center;
	justify-items: start;
	padding: .75rem;
	border-block-end: 1px solid var(--card-border-color);
	cursor: pointer;
	text-align: start;
}

.task-picker__row > span {
	inline-size: 100%;
	text-align: start;
}

.task-picker__row:last-child { border-block-end: 0; }
.task-picker__row:hover { background: var(--grey-100); }

.task-picker__row small {
	display: block;
	color: var(--grey-500);
}

.task-picker__actions {
	display: flex;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: 1.25rem;
}

@media screen and (max-width: $tablet) {
	.task-picker__filters { grid-template-columns: 1fr; }
	.task-picker__row { grid-template-columns: 1.5rem 1fr; }
	.task-picker__row > :nth-child(n+3) { grid-column: 2; }
}
</style>
