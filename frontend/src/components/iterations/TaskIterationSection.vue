<template>
	<div
		v-if="!authStore.isLinkShareAuth"
		class="task-iteration details"
	>
		<h2 class="task-section-title">
			<span class="icon is-grey"><Icon icon="list-check" /></span>
			{{ $t('iteration.taskSection.title') }}
		</h2>
		<div class="field">
			<label class="label">{{ $t('iteration.taskSection.current') }}</label>
			<div class="select is-fullwidth">
				<select
					v-model.number="selectedIterationId"
					:disabled="!canWrite || loading"
					@change="changeIteration"
				>
					<option :value="0">
						{{ $t('iteration.taskSection.none') }}
					</option>
					<option
						v-for="iteration in available"
						:key="iteration.id"
						:value="iteration.id"
					>
						{{ iteration.title }}
					</option>
				</select>
			</div>
			<p class="help">
				{{ $t('iteration.taskSection.hint') }}
			</p>
		</div>
		<div
			v-if="info.history.length"
			class="task-iteration__history"
		>
			<h3>{{ $t('iteration.taskSection.history') }}</h3>
			<ol>
				<li
					v-for="entry in info.history"
					:key="entry.id"
					:class="{'is-current': !entry.leftAt}"
				>
					<strong>{{ entry.iterationTitle }}</strong>
					<span>{{ formatDate(entry.joinedAt) }} · {{ entry.leftAt ? formatDate(entry.leftAt) : $t('iteration.taskSection.current') }}</span>
				</li>
			</ol>
		</div>
	</div>
</template>

<script setup lang="ts">
import {onMounted, reactive, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import {useIterationService} from '@/services/iteration'
import {useAuthStore} from '@/stores/auth'
import {error, success} from '@/message'
import type {IIteration, ITaskIterationInfo} from '@/modelTypes/IIteration'

const props = defineProps<{taskId: number, canWrite: boolean}>()
const {t} = useI18n()
const service = useIterationService()
const authStore = useAuthStore()
const info = reactive<ITaskIterationInfo>({history: []})
const available = ref<IIteration[]>([])
const selectedIterationId = ref(0)
const loading = ref(false)

async function load() {
	loading.value = true
	try {
		const [taskInfo, iterations] = await Promise.all([
			service.getTaskIterations(props.taskId),
			service.list({perPage: 100}),
		])
		Object.assign(info, taskInfo)
		available.value = iterations.items.filter(iteration => iteration.status !== 'closed')
		selectedIterationId.value = taskInfo.current?.id ?? 0
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
}

async function changeIteration() {
	loading.value = true
	try {
		const updated = await service.setTaskIteration(props.taskId, selectedIterationId.value)
		Object.assign(info, updated)
		success({message: t('iteration.taskSection.updated')})
	} catch (e) {
		selectedIterationId.value = info.current?.id ?? 0
		error(e)
	} finally {
		loading.value = false
	}
}

function formatDate(date: Date) {
	return new Intl.DateTimeFormat(undefined, {dateStyle: 'medium'}).format(date)
}

watch(() => props.taskId, load)
onMounted(load)
</script>

<style scoped lang="scss">
.task-iteration__history {
	margin-block-start: 1.5rem;

	ol {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	li {
		border-inline-start: 2px solid var(--grey-300);
		padding: 0 0 1rem 1rem;
	}

	li.is-current { border-color: var(--primary); }

	strong,
	span { display: block; }

	span {
		color: var(--grey-500);
		font-size: .85rem;
	}
}
</style>
