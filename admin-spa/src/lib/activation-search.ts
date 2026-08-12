import { Option, Schema, flow } from 'effect'

const ActivationSearchSchema = Schema.Struct({
  token: Schema.optionalKey(Schema.String),
})

/** Parses the optional activation token from router search input. */
export const parseActivationSearch = flow(
  Schema.decodeUnknownOption(ActivationSearchSchema),
  Option.getOrElse((): typeof ActivationSearchSchema.Type => ({})),
)
