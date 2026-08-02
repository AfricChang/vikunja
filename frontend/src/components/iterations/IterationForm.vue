<template>
	<form
		class="iteration-form"
		@submit.prevent="submit"
	>
		<div class="field">
			<label
				class="label"
				for="iteration-title"
			>{{ $t('iteration.form.title') }}</label>
			<div class="control">
				<input
					id="iteration-title"
					v-model.trim="draft.title"
					class="input"
					required
					maxlength="250"
				>
			</div>
		</div>
		<div class="iteration-form__dates">
			<div class="field">
				<label
					class="label"
					for="iteration-start"
				>{{ $t('iteration.form.start') }}</label>
				<input
					id="iteration-start"
					v-model="startValue"
					class="input"
					type="datetime-local"
					required
				>
			</div>
			<div class="field">
				<label
					class="label"
					for="iteration-end"
				>{{ $t('iteration.form.end') }}</label>
				<input
					id="iteration-end"
					v-model="endValue"
					class="input"
					type="datetime-local"
					required
				>
			</div>
		</div>
		<p
			v-if="invalidDates"
			class="help has-text-danger"
		>
			{{ $t('iteration.form.invalidDates') }}
		</p>
		<div class="field">
			<label
				class="label"
				for="iteration-description"
			>{{ $t('iteration.form.description') }}</label>
			<textarea
				id="iteration-description"
				v-model="draft.description"
				class="textarea"
				rows="4"
				:placeholder="$t('iteration.form.descriptionPlaceholder')"
			/>
		</div>
		<div class="iteration-form__actions">
			<XButton
				variant="tertiary"
				type="button"
				@click="$emit('cancel')"
			>
				{{ $t('misc.cancel') }}
			</XButton>
			<XButton
				type="submit"
				:loading="saving"
				:disabled="invalidDates || !draft.title"
			>
				{{ submitLabel }}
			</XButton>
		</div>
	</form>
</template>

<script setup lang="ts">
import {computed, reactive, ref, watch} from 'vue'

import type {IIteration} from '@/modelTypes/IIteration'

const props = defineProps<{
	iteration?: Partial<IIteration>
	submitLabel: string
	saving?: boolean
}>()

const emit = defineEmits<{
	save: [iteration: Partial<IIteration>]
	cancel: []
}>()

const draft = reactive<Partial<IIteration>>({title: '', description: ''})
const startValue = ref('')
const endValue = ref('')

function localDateValue(value?: Date) {
	if (!value) return ''
	const local = new Date(value.getTime() - value.getTimezoneOffset() * 60000)
	return local.toISOString().slice(0, 16)
}

watch(() => props.iteration, value => {
	Object.assign(draft, {title: value?.title ?? '', description: value?.description ?? ''})
	startValue.value = localDateValue(value?.startDate)
	endValue.value = localDateValue(value?.endDate)
}, {immediate: true, deep: true})

const invalidDates = computed(() => !startValue.value || !endValue.value || new Date(endValue.value) <= new Date(startValue.value))

function submit() {
	if (invalidDates.value || !draft.title) return
	emit('save', {
		...draft,
		startDate: new Date(startValue.value),
		endDate: new Date(endValue.value),
	})
}
</script>

<style scoped lang="scss">
.iteration-form__dates {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 1rem;
}

.iteration-form__actions {
	display: flex;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: 1.5rem;
}

@media screen and (max-width: $tablet) {
	.iteration-form__dates { grid-template-columns: 1fr; }
}
</style>
