<template>
	<div
		class="iterations loader-container"
		:class="{'is-loading': loading}"
	>
		<header class="iterations__header">
			<div>
				<p class="iterations__eyebrow">
					{{ $t('iteration.eyebrow') }}
				</p>
				<h1>{{ $t('iteration.title') }}</h1>
				<p>{{ $t('iteration.subtitle') }}</p>
			</div>
			<XButton
				icon="plus"
				@click="createOpen = true"
			>
				{{ $t('iteration.create') }}
			</XButton>
		</header>
		<Card
			class="iterations__toolbar"
			:padding="false"
		>
			<div class="iterations__toolbar-content">
				<div class="control has-icons-left is-expanded">
					<input
						v-model="query"
						class="input"
						:placeholder="$t('iteration.search')"
						@input="loadDebounced"
					>
					<span class="icon is-left"><Icon icon="search" /></span>
				</div>
				<div class="select">
					<select
						v-model="sortBy"
						@change="load"
					>
						<option value="start_date">
							{{ $t('iteration.sort.start') }}
						</option>
						<option value="end_date">
							{{ $t('iteration.sort.end') }}
						</option>
						<option value="title">
							{{ $t('iteration.sort.title') }}
						</option>
					</select>
				</div>
			</div>
		</Card>
		<section
			v-for="group in groups"
			:key="group.status"
			class="iterations__group"
		>
			<div class="iterations__group-header">
				<h2>{{ group.title }}</h2>
				<span>{{ $t('iteration.count', {count: group.items.length}) }}</span>
			</div>
			<div class="iterations__grid">
				<RouterLink
					v-for="iteration in group.items"
					:key="iteration.id"
					:to="{name: 'iterations.detail', params: {id: iteration.id}}"
					class="iteration-card"
				>
					<div class="iteration-card__top">
						<h3>{{ iteration.title }}</h3>
						<span
							class="iteration-status"
							:class="`is-${iteration.status}`"
						>{{ statusLabel(iteration.status) }}</span>
					</div>
					<p class="iteration-card__dates">
						{{ formatRange(iteration) }}
					</p>
					<div class="iteration-card__progress-label">
						<span>{{ $t('iteration.completedTasks', {done: iteration.doneTaskCount, total: iteration.taskCount}) }}</span><strong>{{ percent(iteration) }}%</strong>
					</div>
					<progress
						class="progress is-small is-success"
						:value="percent(iteration)"
						max="100"
					/>
					<div class="iteration-card__meta">
						<span>{{ $t('iteration.taskCount', {count: iteration.taskCount}) }}</span><span>{{ $t(iteration.status === 'closed' ? 'iteration.viewHistory' : 'iteration.viewDetail') }}</span>
					</div>
					<p
						v-if="iteration.overlapping"
						class="iteration-card__warning"
					>
						{{ $t('iteration.overlapping') }}
					</p>
				</RouterLink>
			</div>
		</section>
		<div
			v-if="!loading && iterations.length === 0"
			class="iterations__empty"
		>
			<h2>{{ $t('iteration.empty.title') }}</h2>
			<p>{{ $t('iteration.empty.text') }}</p>
		</div>
		<Modal
			:enabled="createOpen"
			@close="createOpen = false"
		>
			<Card
				:title="$t('iteration.create')"
				show-close
				@close="createOpen = false"
			>
				<IterationForm
					:iteration="newDefaults"
					:submit-label="$t('iteration.create')"
					:saving="saving"
					@cancel="createOpen = false"
					@save="createIteration"
				/>
			</Card>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue'
import {useI18n} from 'vue-i18n'

import Card from '@/components/misc/Card.vue'
import Modal from '@/components/misc/Modal.vue'
import IterationForm from '@/components/iterations/IterationForm.vue'
import {useIterationService} from '@/services/iteration'
import {useTitle} from '@/composables/useTitle'
import {error, success} from '@/message'
import type {IIteration, IterationStatus} from '@/modelTypes/IIteration'

const {t} = useI18n()
useTitle(() => t('iteration.title'))
const service = useIterationService()
const iterations = ref<IIteration[]>([])
const loading = ref(false)
const saving = ref(false)
const createOpen = ref(false)
const query = ref('')
const sortBy = ref('start_date')
let timer: ReturnType<typeof setTimeout> | null = null

const nextMonday = (() => {
	const date = new Date()
	date.setHours(0, 0, 0, 0)
	date.setDate(date.getDate() + ((8 - date.getDay()) % 7 || 7))
	return date
})()
const newDefaults = computed(() => ({startDate: nextMonday, endDate: new Date(nextMonday.getTime() + 7 * 86400000)}))

const groups = computed(() => ([
	{status: 'active' as const, title: t('iteration.status.active'), items: iterations.value.filter(i => i.status === 'active')},
	{status: 'planned' as const, title: t('iteration.status.planned'), items: iterations.value.filter(i => i.status === 'planned')},
	{status: 'closed' as const, title: t('iteration.status.closed'), items: iterations.value.filter(i => i.status === 'closed')},
]).filter(group => group.items.length > 0))

async function load() {
	loading.value = true
	try {
		iterations.value = (await service.list({q: query.value, sortBy: sortBy.value, orderBy: 'desc', perPage: 100})).items
	} catch (e) { error(e) } finally { loading.value = false }
}

function loadDebounced() { if (timer) clearTimeout(timer); timer = setTimeout(load, 250) }
function statusLabel(status: IterationStatus) { return t(`iteration.status.${status}`) }
function percent(iteration: IIteration) { return Math.round(iteration.completionRate * 100) }
function formatRange(iteration: IIteration) {
	const formatter = new Intl.DateTimeFormat(undefined, {dateStyle: 'medium'})
	return `${formatter.format(iteration.startDate)} — ${formatter.format(iteration.endDate)}`
}

async function createIteration(value: Partial<IIteration>) {
	saving.value = true
	try {
		const created = await service.create(value)
		iterations.value.unshift(created)
		createOpen.value = false
		success({message: t('iteration.created')})
	} catch (e) { error(e) } finally { saving.value = false }
}

onMounted(load)
</script>

<style scoped lang="scss">
.iterations__header {
	display: flex;
	justify-content: space-between;
	align-items: end;
	gap: 1rem;
	margin-block-end: 1.25rem;
}

.iterations__header h1 { margin-block-end: .25rem; }

.iterations__eyebrow {
	text-transform: uppercase;
	letter-spacing: .1em;
	color: var(--grey-500);
	font-size: .75rem;
	margin: 0;
}

.iterations__toolbar-content {
	display: flex;
	gap: .75rem;
	padding: .875rem;
}

.iterations__group { margin-block: 1.75rem 2.5rem; }

.iterations__group-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-block-end: .75rem;
}

.iterations__group-header h2 {
	font-size: 1.25rem;
	margin: 0;
}

.iterations__group-header span {
	color: var(--grey-500);
	font-size: .8rem;
}

.iterations__grid {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 1rem;
}

.iteration-card {
	display: block;
	padding: 1.25rem;
	border: 1px solid var(--card-border-color);
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-sm);
	color: var(--text);
	transition: $transition;
}

.iteration-card:hover {
	transform: translateY(-2px);
	box-shadow: var(--shadow-md);
}

.iteration-card__top,
.iteration-card__progress-label,
.iteration-card__meta {
	display: flex;
	justify-content: space-between;
	gap: .75rem;
}

.iteration-card h3 {
	font-size: 1.15rem;
	margin: 0;
}

.iteration-card__dates,
.iteration-card__meta {
	color: var(--grey-500);
	font-size: .8rem;
}

.iteration-card__progress-label { font-size: .85rem; }

.iteration-status::before {
	content: '';
	inline-size: .5rem;
	block-size: .5rem;
	border-radius: 50%;
	display: inline-block;
	margin-inline-end: .35rem;
	background: var(--grey-500);
}

.iteration-status.is-active::before { background: var(--success); }
.iteration-status.is-planned::before { background: var(--primary); }

.iteration-card__warning {
	color: var(--warning-dark);
	font-size: .8rem;
	margin-block-start: .75rem;
}

.iterations__empty {
	text-align: center;
	padding: 3rem;
	border: 1px dashed var(--card-border-color);
	border-radius: $radius;
}

@media screen and (max-width: $desktop) {
	.iterations__grid { grid-template-columns: repeat(2, 1fr); }
}

@media screen and (max-width: $tablet) {
	.iterations__header,
	.iterations__toolbar-content {
		align-items: stretch;
		flex-direction: column;
	}

	.iterations__grid { grid-template-columns: 1fr; }
}
</style>
