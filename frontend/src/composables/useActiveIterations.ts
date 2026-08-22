import {ref, watch} from 'vue'

import {useIterationService} from '@/services/iteration'
import {useAuthStore} from '@/stores/auth'
import type {IIteration} from '@/modelTypes/IIteration'

// Nothing stops a user from running several iterations at once, but a dashboard
// showing more than a handful stops being a dashboard.
const ACTIVE_ITERATION_LIMIT = 5

/**
 * Loads the iterations the current user has started.
 *
 * `resolved` flips to true as soon as we know whether there is an active
 * iteration — including when the lookup failed or the user is not logged in.
 * Callers gate on it so they can pick between the iteration overview and their
 * default content without first rendering the wrong one.
 */
export function useActiveIterations() {
	const authStore = useAuthStore()
	const service = useIterationService()

	const iterations = ref<IIteration[]>([])
	const resolved = ref(false)

	async function load() {
		// The home page is mounted before unauthenticated users are redirected to
		// the login page, so the request would fail for them.
		if (!authStore.authenticated) {
			iterations.value = []
			resolved.value = true
			return
		}

		try {
			iterations.value = (await service.list({
				status: 'active',
				// The iteration ending soonest is the one that needs attention first.
				sortBy: 'end_date',
				orderBy: 'asc',
				perPage: ACTIVE_ITERATION_LIMIT,
			})).items
		} catch {
			// Never let an iteration lookup keep the caller from rendering its
			// default content — degrade to "no active iteration" instead.
			iterations.value = []
		} finally {
			resolved.value = true
		}
	}

	watch(() => authStore.authenticated, load, {immediate: true})

	return {iterations, resolved}
}
