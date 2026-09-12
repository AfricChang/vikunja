import {describe, it, expect, vi, beforeEach, afterEach} from 'vitest'
import {AxiosError} from 'axios'
import type {AxiosRequestConfig} from 'axios'

import AttachmentService from './attachment'
import BucketService from './bucket'
import {removeToken, refreshToken, saveToken} from '@/helpers/auth'
import type {IAttachment} from '@/modelTypes/IAttachment'
import type {IBucket} from '@/modelTypes/IBucket'

vi.mock('@/helpers/auth', async (importActual) => ({
	...await importActual<typeof import('@/helpers/auth')>(),
	refreshToken: vi.fn(),
}))

function serviceWithBlobResponse(blob: Blob) {
	const service = new AttachmentService()
	service.http = vi.fn().mockResolvedValue({data: blob}) as unknown as typeof service.http
	return service
}

describe('getBlobUrl', () => {
	afterEach(() => {
		vi.restoreAllMocks()
	})

	it('keeps the mime type of the fetched blob', async () => {
		// A blob url without a type downloads instead of rendering when used as iframe src
		const service = serviceWithBlobResponse(new Blob(['%PDF-1.4'], {type: 'application/pdf'}))
		const createObjectURL = vi.spyOn(window.URL, 'createObjectURL').mockReturnValue('blob:mock')

		const url = await service.getBlobUrl({taskId: 1, id: 1} as IAttachment)

		expect(url).toBe('blob:mock')
		const blob = createObjectURL.mock.calls[0][0] as Blob
		expect(blob.type).toBe('application/pdf')
		expect(blob.size).toBeGreaterThan(0)
	})

	it('converts svg blobs to data urls', async () => {
		const service = serviceWithBlobResponse(new Blob(['<svg xmlns="http://www.w3.org/2000/svg"/>'], {type: 'image/svg+xml'}))

		const url = await service.getBlobUrl({taskId: 1, id: 2} as IAttachment)

		expect(url).toMatch(/^data:image\/svg\+xml/)
	})
})

describe('payload transforms on a retried request', () => {
	// A user JWT so the 401 interceptor considers the request refreshable
	const USER_JWT = `x.${btoa(JSON.stringify({id: 1, type: 1}))}.y`

	function bucketServiceFailingOnceWith401() {
		const service = new BucketService()
		const requests: AxiosRequestConfig[] = []

		service.http.defaults.adapter = async (config) => {
			requests.push({...config})
			if (requests.length === 1) {
				const response = {data: {code: 11}, status: 401, statusText: 'Unauthorized', headers: {}, config}
				throw new AxiosError('401', AxiosError.ERR_BAD_REQUEST, config, {}, response)
			}
			return {data: {id: 111, title: 'Doing'}, status: 200, statusText: 'OK', headers: {}, config}
		}

		return {service, requests}
	}

	beforeEach(() => {
		window.API_URL = 'https://api.example.com/api/v1/'
		saveToken(USER_JWT, false)
		vi.mocked(refreshToken).mockResolvedValue(undefined)
	})

	afterEach(() => removeToken())

	it('does not transform an already serialized payload again', async () => {
		const {service, requests} = bucketServiceFailingOnceWith401()

		await service.update({
			id: 111,
			projectId: 26,
			projectViewId: 400,
			title: 'Doing',
			tasks: [],
		} as unknown as IBucket)

		expect(requests).toHaveLength(2)
		expect(requests[1].data).toBe(requests[0].data)
		expect(JSON.parse(requests[1].data as string)).toMatchObject({id: 111, project_id: 26, tasks: []})
	})
})
