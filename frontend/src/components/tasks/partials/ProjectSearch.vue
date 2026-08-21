<template>
	<Multiselect
		class="control is-expanded"
		:placeholder="showAllWhenEmpty ? $t('project.selectOrSearch') : $t('project.search')"
		:search-results="foundProjects"
		label="title"
		:show-empty="showAllWhenEmpty"
		:select-placeholder="$t('project.searchSelect')"
		:model-value="project"
		@update:modelValue="(val) => val === null ? select(null) : Object.assign(project, val)"
		@select="select"
		@search="findProjects"
	>
		<template #searchResult="{option}">
			<span
				v-if="projectStore.getAncestors(option).length > 1"
				class="has-text-grey"
			>
				{{ projectStore.getAncestors(option).filter(p => p.id !== option.id).map(p => getProjectTitle(p)).join(' &gt; ') }} &gt;
			</span>
			{{ getProjectTitle(option) }}
		</template>
	</Multiselect>
</template>

<script lang="ts" setup>
import {reactive, ref, watch} from 'vue'

import type {IProject} from '@/modelTypes/IProject'

import {useProjectStore} from '@/stores/projects'
import {getProjectTitle} from '@/helpers/getProjectTitle'

import ProjectModel from '@/models/project'
import Multiselect from '@/components/input/Multiselect.vue'

const props = withDefaults(defineProps<{
	modelValue?: IProject
	savedFiltersOnly?: boolean
	filter?: (project: IProject) => boolean,
	/**
	 * Lists every project while the search box is empty, so the field can be used as a
	 * plain picker - typing a search becomes optional instead of the only way in.
	 */
	showAllWhenEmpty?: boolean,
}>(), {
	modelValue: () => new ProjectModel(),
	savedFiltersOnly: false,
	filter: () => true,
	showAllWhenEmpty: false,
})

const emit = defineEmits<{
	'update:modelValue': [value: IProject | null]
}>()

const project: IProject = reactive(new ProjectModel())

watch(
	() => props.modelValue,
	(newProject) => Object.assign(project, newProject),
	{
		immediate: true,
		deep: true,
	},
)

const projectStore = useProjectStore()

const searchQuery = ref('')
const foundProjects = ref<IProject[]>([])

function projectsForQuery(query: string): IProject[] {
	// The store's search returns nothing for an empty query, which would leave an
	// untouched box with an empty dropdown - list everything instead, so picking a
	// project doesn't require typing a search first.
	if (props.showAllWhenEmpty && query === '') {
		const all = props.savedFiltersOnly
			? projectStore.savedFilterProjects
			: projectStore.projectsArray.filter(p => p.id > 0 && !p.isArchived)
		return (all as IProject[]).filter(props.filter)
	}

	const found = props.savedFiltersOnly
		? projectStore.searchSavedFilter(query)
		: projectStore.searchProject(query)
	return found.filter(props.filter)
}

function findProjects(query: string) {
	searchQuery.value = query

	if (query === '') {
		select(null)
	}

	foundProjects.value = projectsForQuery(query)
}

// Projects are loaded asynchronously, so the unsearched list has to be refreshed once
// they arrive - otherwise the dropdown stays empty on a freshly loaded page.
watch(
	() => projectStore.projectsArray,
	() => foundProjects.value = projectsForQuery(searchQuery.value),
	{immediate: true},
)

function select(p: IProject | null) {
	if (p === null) {
		Object.assign(project, new ProjectModel())
		emit('update:modelValue', null)
		return
	}
	Object.assign(project, p)
	emit('update:modelValue', project)
}
</script>
