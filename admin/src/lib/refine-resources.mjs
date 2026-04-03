export function getAdminResources() {
  return [
    {
      name: "dashboard",
      list: "/dashboard",
      meta: {
        label: "Teacher Dashboard",
      },
    },
    {
      name: "ai-usage",
      list: "/dashboard/ai-usage",
      meta: {
        label: "AI Usage",
      },
    },
    {
      name: "retrieval-lab",
      list: "/dashboard/retrieval-lab",
      meta: {
        label: "Retrieval Lab",
      },
    },
    {
      name: "students",
      list: "/dashboard",
      show: "/students/:id",
      meta: {
        label: "Students",
      },
    },
    {
      name: "parents",
      list: "/",
      show: "/parents/:id",
      meta: {
        label: "Parents",
      },
    },
  ];
}
