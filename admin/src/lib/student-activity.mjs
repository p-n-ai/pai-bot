function isArray(value) {
  return Array.isArray(value);
}

function toISODate(value) {
  if (typeof value !== "string" || value.length === 0) {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date.toISOString().slice(0, 10);
}

function addDays(isoDate, offset) {
  const date = new Date(`${isoDate}T00:00:00.000Z`);
  date.setUTCDate(date.getUTCDate() + offset);
  return date.toISOString().slice(0, 10);
}

function formatGridLabel(isoDate) {
  const date = new Date(`${isoDate}T00:00:00.000Z`);
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  }).format(date);
}

export function getActivityLevel(count) {
  if (count >= 6) return 4;
  if (count >= 4) return 3;
  if (count >= 2) return 2;
  if (count >= 1) return 1;
  return 0;
}

export function buildStudentActivityGrid(conversations, options = {}) {
  const safeWindowDays = Number.isInteger(options.windowDays) && options.windowDays > 0 ? options.windowDays : 14;
  const timestamps = isArray(conversations)
    ? conversations
        .map((item) => toISODate(item?.timestamp))
        .filter((value) => typeof value === "string")
    : [];

  const anchorDate = toISODate(options.anchorDate) || timestamps.at(-1) || new Date().toISOString().slice(0, 10);
  const counts = timestamps.reduce((result, isoDate) => {
    result[isoDate] = (result[isoDate] || 0) + 1;
    return result;
  }, {});

  return Array.from({ length: safeWindowDays }, (_, index) => {
    const date = addDays(anchorDate, index - (safeWindowDays - 1));
    const count = counts[date] || 0;

    return {
      date,
      shortLabel: formatGridLabel(date),
      count,
      level: getActivityLevel(count),
    };
  });
}
