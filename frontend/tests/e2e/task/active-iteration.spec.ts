import {test, expect} from '../../support/fixtures'
import {Factory} from '../../support/factory'
import {login} from '../../support/authenticateUser'
import {ProjectFactory} from '../../factories/project'
import {UserFactory} from '../../factories/user'
import {createDefaultViews} from '../project/prepareProjects'

const apiV2 = (process.env.API_URL || 'http://127.0.0.1:3456/api/v1')
	.replace(/\/v1\/?$/, '/v2')
	.replace(/\/$/, '')

interface CreatedTask {
	id: number
	title: string
	project_id: number
}

async function clearIterations() {
	for (const table of ['task_iteration_history', 'iteration_tasks', 'iterations']) {
		await Factory.seed(table, [])
	}
}

test.describe('Active iteration after upstream upgrade', () => {
	test.beforeEach(clearIterations)
	test.afterEach(async ({page}) => {
		await page.goto('about:blank')
		await clearIterations()
	})

	test('Shows the active iteration, persists completion and sorts tasks by project without exposing it to another user', async ({authenticatedPage: page, apiContext, currentUser, userToken}) => {
		test.setTimeout(60000)
		const headers = {Authorization: `Bearer ${userToken}`}
		const projects = await ProjectFactory.create(2, {
			owner_id: currentUser.id,
			title: (i: number) => i === 1 ? 'Zulu project' : 'Alpha project',
		})
		const tasks: CreatedTask[] = []
		for (let i = 0; i < projects.length; i++) {
			await createDefaultViews(projects[i].id, i * 4 + 1)
			const response = await apiContext.post(`${apiV2}/projects/${projects[i].id}/tasks/bulk`, {
				headers,
				data: {tasks: [{title: `Iteration work ${i + 1}`}]},
			})
			expect(response.ok()).toBeTruthy()
			const created: {tasks: CreatedTask[]} = await response.json()
			tasks.push(created.tasks[0])
		}

		const now = Date.now()
		const created = await apiContext.post(`${apiV2}/iterations`, {
			headers,
			data: {
				title: 'Upgrade regression iteration',
				start_date: new Date(now - 86400000).toISOString(),
				end_date: new Date(now + 7 * 86400000).toISOString(),
			},
		})
		expect(created.ok()).toBeTruthy()
		const iteration: {id: number, title: string} = await created.json()
		const taskIds = tasks.map(task => task.id)
		const added = await apiContext.post(`${apiV2}/iterations/${iteration.id}/tasks/bulk`, {
			headers,
			data: {task_ids: taskIds},
		})
		expect(added.ok()).toBeTruthy()
		const populated = await apiContext.get(`${apiV2}/iterations/${iteration.id}`, {headers})
		expect(populated.ok()).toBeTruthy()
		expect((await populated.json()).task_count).toBe(2)
		const started = await apiContext.post(`${apiV2}/iterations/${iteration.id}/start`, {headers})
		expect(started.ok()).toBeTruthy()
		expect((await started.json()).status).toBe('active')

		const [otherUser] = await UserFactory.create(1, {id: currentUser.id + 1}, false)
		const {token: otherToken} = await login(null, apiContext, otherUser)
		const otherHeaders = {Authorization: `Bearer ${otherToken}`}
		const deniedRead = await apiContext.get(`${apiV2}/iterations/${iteration.id}/tasks`, {headers: otherHeaders})
		expect(deniedRead.status()).toBe(403)
		const deniedWrite = await apiContext.post(`${apiV2}/iterations/${iteration.id}/tasks/bulk`, {
			headers: otherHeaders,
			data: {task_ids: taskIds},
		})
		expect(deniedWrite.status()).toBe(403)

		await page.goto('/')
		const activeIteration = page.locator('.active-iteration')
		await expect(activeIteration.getByRole('heading', {name: iteration.title})).toBeVisible()
		for (const task of tasks) {
			await expect(activeIteration.locator(`[data-task-id="${task.id}"]`)).toContainText(task.title)
		}
		await expect(activeIteration.locator('progress')).toHaveAttribute('value', '0')
		const completedTask = activeIteration.locator(`[data-task-id="${tasks[1].id}"]`)
		await expect(completedTask.getByRole('checkbox')).toBeEnabled()
		await completedTask.locator('.base-checkbox__label').click()
		await expect(activeIteration.locator('progress')).toHaveAttribute('value', '50')
		await expect(activeIteration.locator('.active-iteration__progress-label strong')).toHaveText('50%')
		await page.reload()
		await expect(activeIteration.locator('progress')).toHaveAttribute('value', '50')
		await expect(activeIteration.locator(`[data-task-id="${tasks[1].id}"]`)).toHaveCount(0)
		await expect(activeIteration.locator(`[data-task-id="${tasks[0].id}"]`)).toContainText(tasks[0].title)

		await activeIteration.getByRole('link', {name: iteration.title, exact: true}).click()
		await expect(page).toHaveURL(new RegExp(`/iterations/${iteration.id}$`))
		const table = page.locator('.iteration-detail__table')
		await expect(table.locator('tbody tr')).toHaveCount(2)
		const projectColumn = table.getByRole('columnheader').filter({has: page.getByRole('button', {name: 'Project', exact: true})})
		const projectButton = projectColumn.getByRole('button')
		for (const [order, expectedTasks] of [
			['asc', [tasks[1], tasks[0]]],
			['desc', [tasks[0], tasks[1]]],
		] as const) {
			const sorted = page.waitForResponse(response => {
				const url = new URL(response.url())
				return url.pathname === `/api/v2/iterations/${iteration.id}/tasks` &&
					url.searchParams.get('sort_by') === 'project' &&
					url.searchParams.get('order_by') === order
			})
			await projectButton.click()
			const response = await sorted
			expect(response.ok()).toBeTruthy()
			const result: {items: CreatedTask[]} = await response.json()
			expect(result.items.map(task => task.id)).toEqual(expectedTasks.map(task => task.id))
			await expect(projectColumn).toHaveAttribute('aria-sort', order === 'asc' ? 'ascending' : 'descending')
			await expect(table.locator('tbody tr td:nth-child(2) strong')).toHaveText(expectedTasks.map(task => task.title))
		}
	})
})
