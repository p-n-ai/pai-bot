import type { ReactNode } from 'react'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export type DataTableCell = {
  key: string
  value: ReactNode
}

export interface DataTableRow {
  cells: Array<DataTableCell>
  id: string
}

export function DataTable({
  className,
  columns,
  rows,
}: {
  className?: string
  columns: Array<string>
  rows: Array<DataTableRow>
}) {
  return (
    <Table className={className}>
      <TableHeader>
        <TableRow>
          {columns.map((column) => (
            <TableHead key={column}>{column}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.id}>
            {row.cells.map((cell) => (
              <TableCell key={cell.key}>{cell.value}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
