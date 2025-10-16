// Validation utilities for the Windows Inventory Agent web console

import Ajv, { ValidateFunction, ErrorObject } from 'ajv'
import addFormats from 'ajv-formats'

// Import schemas
import telemetrySchema from '../schemas/telemetry.schema.json'
import policySchema from '../schemas/policy.schema.json'
import commandSchema from '../schemas/command.schema.json'

// Types
export interface ValidationResult {
  valid: boolean
  errors?: ValidationError[]
}

export interface ValidationError {
  field: string
  message: string
  code?: string
}

// SchemaValidator class for managing JSON schema validation
export class SchemaValidator {
  private ajv: Ajv
  private validators: Map<string, ValidateFunction>

  constructor() {
    this.ajv = new Ajv({
      allErrors: true,
      verbose: true,
      strict: false,
      allowUnionTypes: true,
      useDefaults: true,
      coerceTypes: true,
    })

    // Add common formats
    addFormats(this.ajv)

    // Add custom formats
    this.addCustomFormats()

    this.validators = new Map()
    this.loadSchemas()
  }

  private addCustomFormats(): void {
    // Add custom format validators
    this.ajv.addFormat('hostname', (data: string) => {
      // Basic hostname validation
      const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/
      return hostnameRegex.test(data) && data.length <= 253
    })

    this.ajv.addFormat('semver', (data: string) => {
      // Semantic version validation
      const semverRegex = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/
      return semverRegex.test(data)
    })

    this.ajv.addFormat('uuid', (data: string) => {
      // UUID validation
      const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
      return uuidRegex.test(data)
    })

    this.ajv.addFormat('cron', (data: string) => {
      // Basic cron expression validation (simplified)
      const cronRegex = /^(\*|([0-9]|1[0-9]|2[0-3]|3[0-1]|4[0-9]|5[0-9]|6[0-1])|\*\/[0-9]+|([0-9]+-[0-9]+)|([0-9]+(,[0-9]+)*))\s+(\*|([0-9]|1[0-9]|2[0-3]|4[0-9]|5[0-9]|6[0-1])|\*\/[0-9]+|([0-9]+-[0-9]+)|([0-9]+(,[0-9]+)*))\s+(\*|([0-9]|1[0-9]|2[0-3]|3[0-1]|4[0-9]|5[0-9]|6[0-1])|\*\/[0-9]+|([0-9]+-[0-9]+)|([0-9]+(,[0-9]+)*))\s+(\*|([0-9]|1[0-9]|2[0-3]|3[0-1]|4[0-9]|5[0-9]|6[0-1]|1[0-2]|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)|\*\/[0-9]+|([0-9]+-[0-9]+)|([0-9]+(,[0-9]+)*))\s+(\*|([0-9]|1[0-9]|2[0-3]|3[0-1]|4[0-9]|5[0-9]|6[0-1]|1[0-2]|SUN|MON|TUE|WED|THU|FRI|SAT)|\*\/[0-9]+|([0-9]+-[0-9]+)|([0-9]+(,[0-9]+)*))$/
      return cronRegex.test(data)
    })
  }

  private loadSchemas(): void {
    // Compile and store validators
    this.validators.set('telemetry', this.ajv.compile(telemetrySchema))
    this.validators.set('policy', this.ajv.compile(policySchema))
    this.validators.set('command', this.ajv.compile(commandSchema))
  }

  // Validate data against a named schema
  validate(schemaName: string, data: any): ValidationResult {
    const validator = this.validators.get(schemaName)
    if (!validator) {
      return {
        valid: false,
        errors: [{
          field: 'root',
          message: `Schema '${schemaName}' not found`,
          code: 'SCHEMA_NOT_FOUND'
        }]
      }
    }

    const valid = validator(data)
    if (valid) {
      return { valid: true }
    }

    const errors = validator.errors?.map(this.convertAjvError) || []
    return { valid: false, errors }
  }

  // Validate telemetry data
  validateTelemetry(data: any): ValidationResult {
    return this.validate('telemetry', data)
  }

  // Validate policy data
  validatePolicy(data: any): ValidationResult {
    return this.validate('policy', data)
  }

  // Validate command data
  validateCommand(data: any): ValidationResult {
    return this.validate('command', data)
  }

  private convertAjvError(error: ErrorObject): ValidationError {
    return {
      field: error.instancePath || 'root',
      message: error.message || 'Validation error',
      code: error.keyword || 'UNKNOWN_ERROR'
    }
  }
}

// Singleton validator instance
let validatorInstance: SchemaValidator | null = null

// Get the global validator instance
export function getValidator(): SchemaValidator {
  if (!validatorInstance) {
    validatorInstance = new SchemaValidator()
  }
  return validatorInstance
}

// Convenience validation functions
export function validateTelemetry(data: any): ValidationResult {
  return getValidator().validateTelemetry(data)
}

export function validatePolicy(data: any): ValidationResult {
  return getValidator().validatePolicy(data)
}

export function validateCommand(data: any): ValidationResult {
  return getValidator().validateCommand(data)
}

// Async validation wrapper for consistency with Go API
export async function validateTelemetryAsync(data: any): Promise<ValidationResult> {
  return Promise.resolve(validateTelemetry(data))
}

export async function validatePolicyAsync(data: any): Promise<ValidationResult> {
  return Promise.resolve(validatePolicy(data))
}

export async function validateCommandAsync(data: any): Promise<ValidationResult> {
  return Promise.resolve(validateCommand(data))
}