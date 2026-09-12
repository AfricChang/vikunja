// The api sends this year instead of null for an unset date.
const ZERO_TIME_YEAR = 1

export function parseDateOrNull(date: Date | string | null | undefined): Date | null {
	const parsed = date instanceof Date
		? date
		: (typeof date === 'string' && date !== '' ? new Date(date) : null)

	if (
		parsed === null ||
		Number.isNaN(parsed.getTime()) ||
		parsed.getUTCFullYear() <= ZERO_TIME_YEAR
	) {
		return null
	}

	return parsed
}
