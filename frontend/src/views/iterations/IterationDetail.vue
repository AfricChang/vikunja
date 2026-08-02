<template>
	<div
		class="iteration-detail loader-container"
		:class="{'is-loading': loading}"
	>
		<header class="iteration-detail__header">
			<div>
				<p class="iteration-detail__eyebrow">
					{{ statusLabel }}
				</p>
				<h1>{{ iteration?.title }}</h1>
				<p>{{ formatRange }}</p>
			</div>
			<div class="iteration-detail__actions">
				<XButton
					v-if="iteration?.status === 'planned'"
					variant="secondary"
					@click="start"
				>
					{{ $t('iteration.start') }}
				</XButton>
				<XButton
					v-if="iteration?.status === 'closed'"
					variant="secondary"
					@click="reopen"
				>
					{{ $t('iteration.reopen') }}
				</XButton>
				<XButton
					v-if="iteration && iteration.status !== 'closed'"
					variant="secondary"
					@click="formOpen = true"
				>
					{{ $t('misc.edit') }}
				</XButton>
				<XButton
					v-if="iteration && iteration.status !== 'closed'"
					variant="secondary"
					@click="nextOpen = true"
				>
					{{ $t('iteration.next.create') }}
				</XButton>
				<XButton
					v-if="iteration && iteration.status !== 'closed'"
					@click="closeOpen = true"
				>
					{{ $t('iteration.closeAction') }}
				</XButton>
			</div>
		</header>
		<Card
			v-if="iteration"
			class="iteration-detail__summary"
		>
			<div class="iteration-detail__metrics">
				<div><span>{{ $t('iteration.metrics.total') }}</span><strong>{{ iteration.taskCount }}</strong></div>
				<div><span>{{ $t('iteration.metrics.done') }}</span><strong>{{ iteration.doneTaskCount }}</strong></div>
				<div><span>{{ $t('iteration.metrics.open') }}</span><strong>{{ iteration.taskCount - iteration.doneTaskCount }}</strong></div>
				<div><span>{{ $t('iteration.metrics.rate') }}</span><strong>{{ percent }}%</strong></div>
			</div>
		</Card>
		<div
			v-if="iteration?.overlapping"
			class="notification is-warning is-light"
		>
			{{ $t('iteration.overlapping') }}
		</div>
		<Card
			class="iteration-detail__toolbar"
			:padding="false"
		>
			<div class="iteration-detail__toolbar-content">
				<div class="control has-icons-left is-expanded">
					<input
						v-model="query"
						class="input"
						:placeholder="$t('iteration.tasks.search')"
						@input="loadTasksDebounced"
					>
					<span class="icon is-left"><Icon icon="search" /></span>
				</div>
				<div class="select">
					<select
						v-model="status"
						@change="loadTasks"
					>
						<option value="">
							{{ $t('iteration.tasks.allStatus') }}
						</option><option value="open">
							{{ $t('iteration.tasks.open') }}
						</option><option value="done">
							{{ $t('task.attributes.done') }}
						</option>
					</select>
				</div>
				<XButton
					icon="plus"
					@click="pickerOpen = true"
				>
					{{ $t('iteration.tasks.add') }}
				</XButton>
			</div>
		</Card>
		<div
			class="iteration-detail__bulk"
			:class="{'is-visible': selected.length > 0}"
		>
			<span>{{ $t('iteration.tasks.selected', {count: selected.length}) }}</span>
			<XButton
				variant="tertiary"
				:disabled="selected.length === 0"
				@click="removeSelected"
			>
				{{ $t('iteration.tasks.remove') }}
			</XButton>
		</div>
		<Card
			class="iteration-detail__table"
			:padding="false"
		>
			<table class="table is-fullwidth is-hoverable">
				<thead>
					<tr>
						<th>
							<input
								v-model="selectAll"
								type="checkbox"
							>
						</th><th>{{ $t('task.title') }}</th><th>{{ $t('task.attributes.project') }}</th><th>{{ $t('task.attributes.status') }}</th><th>{{ $t('task.attributes.priority') }}</th><th>{{ $t('task.attributes.dueDate') }}</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="task in tasks"
						:key="task.id"
					>
						<td>
							<input
								v-model="selected"
								type="checkbox"
								:value="task.id"
							>
						</td>
						<td>
							<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">
								<strong>{{ task.title }}</strong><small class="is-block has-text-grey">{{ task.identifier || `#${task.id}` }}</small>
							</RouterLink>
						</td>
						<td>{{ projectTitle(task.projectId) }}</td>
						<td>{{ task.done ? $t('task.attributes.done') : $t('iteration.tasks.open') }}</td>
						<td>{{ task.priority || '—' }}</td>
						<td>{{ task.dueDate ? formatDate(task.dueDate) : '—' }}</td>
					</tr>
					<tr v-if="!loading && tasks.length === 0">
						<td
							colspan="6"
							class="has-text-centered p-6"
						>
							{{ $t('iteration.tasks.empty') }}
						</td>
					</tr>
				</tbody>
			</table>
		</Card>
		<Modal
			:enabled="pickerOpen"
			:wide="true"
			@close="pickerOpen = false"
		>
			<Card
				:title="$t('iteration.tasks.add')"
				show-close
				@close="pickerOpen = false"
			>
				<TaskPicker
					:saving="saving"
					@cancel="pickerOpen = false"
					@add="addTasks"
				/>
			</Card>
		</Modal>
		<Modal
			:enabled="formOpen"
			@close="formOpen = false"
		>
			<Card
				:title="$t('misc.edit')"
				show-close
				@close="formOpen = false"
			>
				<IterationForm
					:iteration="iteration ?? undefined"
					:submit-label="$t('misc.save')"
					:saving="saving"
					@cancel="formOpen = false"
					@save="updateIteration"
				/>
			</Card>
		</Modal>
		<Modal
			:enabled="nextOpen"
			@close="nextOpen = false"
		>
			<Card
				:title="$t('iteration.next.title')"
				show-close
				@close="nextOpen = false"
			>
				<IterationForm
					:iteration="nextDefaults"
					:submit-label="$t('iteration.next.create')"
					:saving="saving"
					@cancel="nextOpen = false"
					@save="createNext"
				/>
			</Card>
		</Modal>
		<Modal
			:enabled="closeOpen"
			@close="closeOpen = false"
		>
			<Card
				:title="$t('iteration.closeAction')"
				show-close
				@close="closeOpen = false"
			>
				<div class="field">
					<label class="label">{{ $t('iteration.close.target') }}</label>
					<div class="select is-fullwidth">
						<select v-model.number="targetId">
							<option :value="0">
								{{ $t('iteration.close.noRollover') }}
							</option><option
								v-for="item in availableTargets"
								:key="item.id"
								:value="item.id"
							>
								{{ item.title }}
							</option>
						</select>
					</div>
				</div>
				<label class="checkbox"><input
					v-model="rolloverAll"
					type="checkbox"
				> {{ $t('iteration.close.selectAll') }}</label>
				<div class="iteration-detail__close-list">
					<label
						v-for="task in openTasks"
						:key="task.id"
						class="checkbox"
					><input
						v-model="rolloverIds"
						type="checkbox"
						:value="task.id"
					> {{ task.title }}</label>
				</div>
				<p class="help">
					{{ $t('iteration.close.hint') }}
				</p>
				<div class="iteration-detail__modal-actions">
					<XButton
						variant="tertiary"
						@click="closeOpen = false"
					>
						{{ $t('misc.cancel') }}
					</XButton><XButton
						:loading="saving"
						@click="closeIteration"
					>
						{{ $t('iteration.close.submit') }}
					</XButton>
				</div>
			</Card>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import Card from '@/components/misc/Card.vue'
import Modal from '@/components/misc/Modal.vue'
import IterationForm from '@/components/iterations/IterationForm.vue'
import TaskPicker from '@/components/iterations/TaskPicker.vue'
import {useIterationService} from '@/services/iteration'
import {useProjectStore} from '@/stores/projects'
import {useTitle} from '@/composables/useTitle'
import {error, success} from '@/message'
import type {IIteration} from '@/modelTypes/IIteration'
import type {ITask} from '@/modelTypes/ITask'

const {t} = useI18n()
const route = useRoute()
const router = useRouter()
const service = useIterationService()
const projectStore = useProjectStore()
const iteration = ref<IIteration | null>(null)
const tasks = ref<ITask[]>([])
const loading = ref(false)
const saving = ref(false)
const query = ref('')
const status = ref('')
const selected = ref<number[]>([])
const pickerOpen = ref(false)
const formOpen = ref(false)
const nextOpen = ref(false)
const closeOpen = ref(false)
const targetId = ref(0)
const rolloverIds = ref<number[]>([])
const rolloverAll = ref(true)
let timer: ReturnType<typeof setTimeout> | null = null

useTitle(() => iteration.value?.title ?? t('iteration.title'))
const statusLabel = computed(() => iteration.value ? t(`iteration.status.${iteration.value.status}`) : '')
const percent = computed(() => iteration.value ? Math.round(iteration.value.completionRate * 100) : 0)
const formatRange = computed(() => iteration.value ? `${formatDate(iteration.value.startDate)} — ${formatDate(iteration.value.endDate)}` : '')
const availableTargets = ref<IIteration[]>([])
const openTasks = computed(() => tasks.value.filter(task => !task.done))
const selectAll = computed({
	get: () => tasks.value.length > 0 && selected.value.length === tasks.value.length,
	set: value => { selected.value = value ? tasks.value.map(task => task.id) : [] },
})
const nextDefaults = computed(() => iteration.value ? {title: t('iteration.next.defaultTitle', {title: iteration.value.title}), startDate: iteration.value.endDate, endDate: new Date(iteration.value.endDate.getTime() + (iteration.value.endDate.getTime() - iteration.value.startDate.getTime()))} : {})

function projectTitle(id: number) { return projectStore.projects[id]?.title ?? `#${id}` }
function formatDate(date: Date) { return new Intl.DateTimeFormat(undefined, {dateStyle: 'medium'}).format(date) }

async function load() {
	loading.value = true
	try {
		const id = Number(route.params.id)
		const [loaded, taskResult, targetResult] = await Promise.all([
			service.get(id),
			service.listTasks(id, {q: query.value, status: status.value, sortBy: 'done', orderBy: 'asc', perPage: 100}),
			service.list({status: 'planned', perPage: 100}),
		])
		iteration.value = loaded
		tasks.value = taskResult.items
		availableTargets.value = targetResult.items.filter(item => item.id !== id && item.status !== 'closed')
		if (rolloverAll.value) rolloverIds.value = openTasks.value.map(task => task.id)
	} catch (e) { error(e) } finally { loading.value = false }
}

function loadTasksDebounced() { if (timer) clearTimeout(timer); timer = setTimeout(load, 250) }
function loadTasks() { load() }

async function addTasks(taskIds: number[]) {
	if (!iteration.value) return
	saving.value = true
	try {
		await service.addTasks(iteration.value.id, taskIds)
		pickerOpen.value = false
		await load()
		success({message: t('iteration.tasks.added')})
	} catch (e) { error(e) } finally { saving.value = false }
}

async function removeSelected() {
	if (!iteration.value || selected.value.length === 0) return
	saving.value = true
	try {
		await service.removeTasks(iteration.value.id, selected.value)
		selected.value = []
		await load()
		success({message: t('iteration.tasks.removed')})
	} catch (e) { error(e) } finally { saving.value = false }
}

async function start() { if (!iteration.value) return; try { iteration.value = await service.start(iteration.value.id); success({message: t('iteration.started')}) } catch (e) { error(e) } }
async function reopen() { if (!iteration.value) return; try { iteration.value = await service.reopen(iteration.value.id); await load(); success({message: t('iteration.reopened')}) } catch (e) { error(e) } }
async function updateIteration(value: Partial<IIteration>) { if (!iteration.value) return; saving.value = true; try { iteration.value = await service.update({...value, id: iteration.value.id}); formOpen.value = false; success({message: t('iteration.updated')}) } catch (e) { error(e) } finally { saving.value = false } }
async function createNext(value: Partial<IIteration>) { if (!iteration.value) return; saving.value = true; try { const created = await service.createNext(iteration.value.id, value); nextOpen.value = false; await router.push({name: 'iterations.detail', params: {id: created.id}}) } catch (e) { error(e) } finally { saving.value = false } }
async function closeIteration() { if (!iteration.value) return; saving.value = true; try { await service.close(iteration.value.id, targetId.value, rolloverAll.value ? openTasks.value.map(task => task.id) : rolloverIds.value); closeOpen.value = false; await load(); success({message: t('iteration.closed')}) } catch (e) { error(e) } finally { saving.value = false } }

watch(() => route.params.id, load)
onMounted(load)
</script>

<style scoped lang="scss">
.iteration-detail__header {
	display: flex;
	justify-content: space-between;
	align-items: end;
	gap: 1rem;
	margin-block-end: 1.25rem;
}

.iteration-detail__header h1 { margin-block-end: .25rem; }

.iteration-detail__eyebrow {
	text-transform: uppercase;
	color: var(--grey-500);
	font-size: .75rem;
	letter-spacing: .1em;
}

.iteration-detail__actions {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
}

.iteration-detail__metrics {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1px;
	background: var(--card-border-color);
}

.iteration-detail__metrics > div {
	background: var(--white);
	padding: 1rem;
}

.iteration-detail__metrics span, .iteration-detail__metrics strong { display: block; }

.iteration-detail__metrics span {
	color: var(--grey-500);
	font-size: .75rem;
	text-transform: uppercase;
}

.iteration-detail__metrics strong {
	font-size: 1.5rem;
	margin-block-start: .25rem;
}

.iteration-detail__toolbar-content {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .875rem;
}

.iteration-detail__bulk {
	display: none;
	align-items: center;
	justify-content: space-between;
	background: var(--text-strong);
	color: var(--white);
	border-radius: $radius;
	padding: .5rem .75rem;
	margin-block-end: .75rem;
}

.iteration-detail__bulk.is-visible { display: flex; }
.iteration-detail__table { overflow: hidden; }
.iteration-detail__table table { margin: 0; }

.iteration-detail__close-list {
	max-block-size: 15rem;
	overflow: auto;
	display: grid;
	gap: .5rem;
	padding-block: .75rem;
}

.iteration-detail__modal-actions {
	display: flex;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: 1.25rem;
}

@media screen and (max-width: $tablet) {
	.iteration-detail__header,
	.iteration-detail__toolbar-content {
		align-items: stretch;
		flex-direction: column;
	}

	.iteration-detail__metrics { grid-template-columns: repeat(2, 1fr); }
	.iteration-detail__table { overflow-x: auto; }
}
</style>
