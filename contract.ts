import { initContract } from "@ts-rest/core";
import { method } from "lodash";
import { z } from "zod";

const c = initContract();

export const genericMessageDataSchema = z
  .object({
    message: z.string(),
    details: z.object({}).passthrough().optional(),
  })
  .strict();

export const resultDataSchema = z
  .object({
    id: z.string(),
    result: z.object({}).passthrough(),
  })
  .strict();

export const learningSchema = z.object({
  summary: z
    .string()
    .describe(
      "The new information that was learned. Be generic, do not refer to the entities.",
    ),
  entities: z
    .array(
      z.object({
        name: z
          .string()
          .describe("The name of the entity this learning relates to."),
        type: z.enum(["tool"]),
      }),
    )
    .describe("The entities this learning relates to."),
  relevance: z.object({
    temporality: z
      .enum(["transient", "persistent"])
      .describe("How long do you expect this learning to be relevant for."),
  }),
});

export const agentDataSchema = z
  .object({
    done: z.boolean().optional(),
    result: z.any().optional(),
    summary: z.string().optional(),
    learnings: z.array(learningSchema).optional(),
    issue: z.string().optional(),
    invocations: z
      .array(
        z.object({
          id: z.string().optional(),
          toolName: z.string(),
          reasoning: z.string(),
          input: z.object({}).passthrough(),
        }),
      )
      .optional(),
  })
  .strict();

export const messageDataSchema = z.union([
  resultDataSchema,
  agentDataSchema,
  genericMessageDataSchema,
]);

export const FunctionConfigSchema = z.object({
  cache: z
    .object({
      keyPath: z.string(),
      ttlSeconds: z.number(),
    })
    .optional(),
  retryCountOnStall: z.number().optional(),
  timeoutSeconds: z.number().optional(),
  executionIdPath: z.string().optional(),
  requiresApproval: z.boolean().default(false).optional(),
  private: z.boolean().default(false).optional(),
});

export const definition = {
  registerMachine: {
    method: "POST",
    path: "/machines",
    headers: z.object({
      authorization: z.string(),
      "x-machine-id": z.string(),
      "x-machine-sdk-version": z.string(),
      "x-machine-sdk-language": z.string(),
    }),
    body: z.object({
      service: z.string(),
      functions: z
        .array(
          z.object({
            name: z.string(),
            description: z.string().optional(),
            schema: z.string(),
            config: FunctionConfigSchema.optional(),
          }),
        )
        .optional(),
    }),
    responses: {
      200: z.object({
        queueUrl: z.string(),
        region: z.string(),
        enabled: z.boolean().default(true),
        expiration: z.date(),
        credentials: z.object({
          accessKeyId: z.string(),
          secretAccessKey: z.string(),
          sessionToken: z.string(),
        }),
      }),
      204: z.undefined(),
    },
  },
  acknowledgeJob: {
    method: "PUT",
    path: "/jobs/:jobId",
    headers: z.object({
      authorization: z.string(),
      "x-machine-id": z.string(),
      "x-machine-sdk-version": z.string(),
      "x-machine-sdk-language": z.string(),
    }),
    pathParams: z.object({
      jobId: z.string(),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
    body: z.undefined(),
  },
  persistJobResult: {
    method: "POST",
    path: "/jobs/:jobId/result",
    headers: z.object({
      authorization: z.string(),
      "x-machine-id": z.string(),
      "x-machine-sdk-version": z.string(),
      "x-machine-sdk-language": z.string(),
    }),
    pathParams: z.object({
      jobId: z.string(),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
    body: z.object({
      result: z.string(),
      resultType: z.enum(["resolution", "rejection"]),
      // TODO: wrap this in meta
      functionExecutionTime: z.number().optional(),
    }),
  },
  live: {
    method: "GET",
    path: "/live",
    responses: {
      200: z.object({
        status: z.string(),
      }),
    },
  },
  getContract: {
    method: "GET",
    path: "/contract",
    responses: {
      200: z.object({
        contract: z.string(),
      }),
    },
  },
  // management routes
  getClustersForUser: {
    method: "GET",
    path: "/clusters",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          name: z.string(),
          apiSecret: z.string(),
          createdAt: z.date(),
          description: z.string().nullable(),
          additionalContext: z
            .object({
              current: z
                .object({
                  version: z.string(),
                  content: z.string(),
                })
                .describe("Current cluster context version"),
              history: z
                .array(
                  z.object({
                    version: z.string(),
                    content: z.string(),
                  }),
                )
                .describe("History of the cluster context versions"),
            })
            .nullable()
            .describe(
              "Additional cluster context which is included in all workflows",
            ),
        }),
      ),
      401: z.undefined(),
    },
  },
  createClusterForUser: {
    method: "POST",
    path: "/clusters",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      204: z.undefined(),
    },
    body: z.object({
      description: z
        .string()
        .describe("Human readable description of the cluster"),
    }),
  },
  editClusterDetails: {
    method: "PUT",
    path: "/clusters/:clusterId",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
    body: z.object({
      name: z.string(),
      description: z.string(),
      additionalContext: z
        .object({
          current: z
            .object({
              version: z.string(),
              content: z.string(),
            })
            .describe("Current cluster context version"),
          history: z
            .array(
              z.object({
                version: z.string(),
                content: z.string(),
              }),
            )
            .describe("History of the cluster context versions"),
        })
        .optional()
        .describe(
          "Additional cluster context which is included in all workflows",
        ),
      debug: z
        .boolean()
        .optional()
        .describe(
          "Enable additional logging (Including prompts and results) for use by Inferable support",
        ),
    }),
  },
  getClusterDetailsForUser: {
    method: "GET",
    path: "/clusters/:clusterId",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.object({
        id: z.string(),
        name: z.string(),
        description: z.string().nullable(),
        apiSecret: z.string(),
        createdAt: z.date(),
        additionalContext: z
          .object({
            current: z.object({
              version: z.string(),
              content: z.string(),
            }),
            history: z.array(
              z.object({
                version: z.string(),
                content: z.string(),
              }),
            ),
          })
          .nullable(),
        debug: z.boolean().default(false),
        machines: z.array(
          z.object({
            id: z.string(),
            description: z.string().optional(), // TODO: deprecate
            lastPingAt: z.date().nullable(),
            ip: z.string().nullable(),
          }),
        ),
        jobs: z.array(
          z.object({
            id: z.string(),
            targetFn: z.string(),
            service: z.string().nullable(),
            status: z.string(),
            resultType: z.string().nullable(),
            createdAt: z.date(),
            functionExecutionTime: z.number().nullable(),
          }),
        ),
        definitions: z.array(
          z.object({
            name: z.string(),
            description: z.string().optional(),
            functions: z
              .array(
                z.object({
                  name: z.string(),
                  description: z.string().optional(),
                  schema: z.string().optional(),
                }),
              )
              .optional(),
          }),
        ),
      }),
      401: z.undefined(),
      404: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
    }),
  },
  getClusterServiceDetailsForUser: {
    method: "GET",
    path: "/clusters/:clusterId/service/:serviceName",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.object({
        jobs: z.array(
          z.object({
            id: z.string(),
            targetFn: z.string(),
            service: z.string().nullable(),
            status: z.string(),
            resultType: z.string().nullable(),
            createdAt: z.date(),
            functionExecutionTime: z.number().nullable(),
          }),
        ),
        definition: z
          .object({
            name: z.string(),
            functions: z
              .array(
                z.object({
                  name: z.string(),
                  rate: z
                    .object({
                      per: z.enum(["minute", "hour"]),
                      limit: z.number(),
                    })
                    .optional(),
                  cacheTTL: z.number().optional(),
                }),
              )
              .optional(),
          })
          .nullable(),
      }),
      401: z.undefined(),
      404: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
      serviceName: z.string(),
    }),
    query: z.object({
      limit: z.coerce.number().min(100).max(5000).default(2000),
    }),
  },
  getActivityForCluster: {
    method: "GET",
    path: "/clusters/:clusterId/events",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          type: z.string(),
          machineId: z.string().nullable(),
          service: z.string().nullable(),
          createdAt: z.string(),
          jobId: z.string().nullable(),
          targetFn: z.string().nullable(),
          resultType: z.string().nullable(),
          status: z.string().nullable(),
          workflowId: z.string().nullable(),
          meta: z.any().nullable(),
          id: z.string(),
        }),
      ),
      401: z.undefined(),
      404: z.undefined(),
    },
    query: z.object({
      type: z.string().optional(),
      jobId: z.string().optional(),
      machineId: z.string().optional(),
      service: z.string().optional(),
      workflowId: z.string().optional(),
      includeMeta: z.string().optional(),
    }),
  },
  getMetaForActivity: {
    method: "GET",
    path: "/clusters/:clusterId/events/:eventId/meta",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.object({
        type: z.string(),
        machineId: z.string().nullable(),
        service: z.string().nullable(),
        createdAt: z.string(),
        jobId: z.string().nullable(),
        targetFn: z.string().nullable(),
        resultType: z.string().nullable(),
        status: z.string().nullable(),
        meta: z.unknown(),
        id: z.string(),
      }),
      401: z.undefined(),
      404: z.undefined(),
    },
  },
  executeJobSync: {
    method: "POST",
    path: "/clusters/:clusterId/execute",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      service: z.string(),
      function: z.string(),
      input: z.object({}).passthrough(),
    }),
    responses: {
      401: z.undefined(),
      404: z.undefined(),
      200: z.object({
        resultType: z.string(),
        result: z.any(),
        status: z.string(),
      }),
      400: z.object({
        message: z.string(),
      }),
      500: z.object({
        error: z.string(),
      }),
    },
  },
  createWorkflow: {
    method: "POST",
    path: "/clusters/:clusterId/workflows",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      message: z
        .string()
        .optional()
        .describe("The workflow prompt, do not provide if using a template"),
      test: z
        .object({
          enabled: z.boolean().default(false),
          mocks: z
            .record(
              z.object({
                output: z
                  .object({})
                  .passthrough()
                  .describe("The mock output of the function"),
              }),
            )
            .optional()
            .describe(
              "Function mocks to be used in the workflow. (Keys should be function in the format <SERVICE>_<FUNCTION>)",
            ),
        })
        .optional()
        .describe(
          "When provided, the workflow will be run as a test / evaluation",
        ),
      template: z
        .object({
          id: z.string().describe("The template ID"),
          input: z
            .object({})
            .passthrough()
            .describe(
              "The input arguments, these should match what is described in the template definition",
            ),
        })
        .optional()
        .describe("A template which the workflow should be created from"),
    }),
    responses: {
      201: z.object({
        id: z.string().describe("The id of the newly created workflow"),
      }),
      401: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
    }),
  },
  deleteWorkflow: {
    method: "DELETE",
    path: "/clusters/:clusterId/workflows/:workflowId",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.undefined(),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
    pathParams: z.object({
      workflowId: z.string(),
      clusterId: z.string(),
    }),
  },
  addMessageToWorkflow: {
    method: "POST",
    path: "/clusters/:clusterId/workflows/:workflowId/messages",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      id: z
        .string()
        .length(26)
        .regex(/^[0-9a-z]+$/i),
      message: z.string(),
    }),
    responses: {
      200: z.object({
        messages: z.array(
          z.object({
            id: z.string(),
            data: messageDataSchema,
            type: z.enum([
              "human",
              "template",
              "result",
              "agent",
              "agent-invalid",
              "supervisor",
            ]),
            createdAt: z.date(),
            pending: z.boolean().default(false),
          }),
        ),
      }),
      401: z.undefined(),
    },
    pathParams: z.object({
      workflowId: z.string(),
      clusterId: z.string(),
    }),
  },
  getWorkflowMessages: {
    method: "GET",
    path: "/clusters/:clusterId/workflows/:workflowId/messages",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          data: messageDataSchema,
          type: z.enum([
            "human",
            "template",
            "result",
            "agent",
            "agent-invalid",
            "supervisor",
          ]),
          createdAt: z.date(),
          pending: z.boolean().default(false),
          displayableContext: z.record(z.string()).nullable(),
        }),
      ),
      401: z.undefined(),
    },
  },
  getClusterWorkflows: {
    method: "GET",
    path: "/clusters/:clusterId/workflows",
    headers: z.object({
      authorization: z.string(),
    }),
    query: z.object({
      userId: z.string().optional(),
      test: z.coerce.string().transform((value) => value === "true"),
      limit: z.coerce.number().min(10).max(50).default(50),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          name: z.string(),
          userId: z.string().nullable(),
          createdAt: z.date(),
          status: z.string().nullable(),
          args: z.record(z.string()).nullable(),
          parentWorkflowId: z.string().nullable(),
          test: z.boolean(),
        }),
      ),
      401: z.undefined(),
    },
  },
  createWorkflowTemplate: {
    method: "POST",
    path: "/clusters/:clusterId/templates",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      name: z.string(),
      description: z.string().optional(),
      instructions: z.string(),
      actions: z
        .array(z.object({ label: z.string(), prompt: z.string() }))
        .optional(),
      inputs: z.array(z.string()).optional(),
      trigger: z.enum(["manual", "webhook", "app"]),
    }),
    responses: {
      201: z.object({
        id: z.string(),
        version: z.number(),
      }),
      400: z.undefined(),
      401: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
    }),
  },
  getTemplateParameters: {
    method: "GET",
    path: "/clusters/:clusterId/template-parameters",
    headers: z.object({
      authorization: z.string(),
    }),
    query: z.object({
      workflowId: z.string(),
      messageId: z.string().optional(),
    }),
    responses: {
      200: z.object({
        name: z.string(),
        description: z.string().nullable(),
        instructions: z.string(),
        actions: z.array(z.object({ label: z.string(), prompt: z.string() })),
        inputs: z.array(z.string()),
      }),
      401: z.undefined(),
    },
  },
  updateWorkflowTemplate: {
    method: "PUT",
    path: "/clusters/:clusterId/templates/:templateId",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      name: z.string(),
      instructions: z.string(),
      description: z.string().optional(),
      actions: z
        .array(z.object({ label: z.string(), prompt: z.string() }))
        .optional(),
      inputs: z.array(z.string()).optional(),
      trigger: z.enum(["manual", "webhook", "app"]),
    }),
    responses: {
      200: z.object({
        version: z.number(),
      }),
    },
    pathParams: z.object({
      clusterId: z.string(),
      templateId: z.string(),
    }),
  },
  deleteWorkflowTemplate: {
    method: "DELETE",
    path: "/clusters/:clusterId/templates/:templateId",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.undefined(),
    responses: {
      204: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
      templateId: z.string(),
    }),
  },
  getWorkflowTemplates: {
    method: "GET",
    path: "/clusters/:clusterId/templates",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          name: z.string(),
          instructions: z.string(),
          description: z.string().nullable(),
          inputs: z.array(z.string()),
          trigger: z.enum(["manual", "webhook", "app"]),
          version: z.number(),
          createdAt: z.date(),
        }),
      ),
      401: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
    }),
  },
  getWorkflowTemplate: {
    method: "GET",
    path: "/clusters/:clusterId/templates/:templateId",
    headers: z.object({
      authorization: z.string(),
    }),
    query: z.object({
      version: z.string().optional(),
    }),
    responses: {
      200: z.object({
        template: z.object({
          id: z.string(),
          name: z.string(),
          instructions: z.string(),
          description: z.string().nullable(),
          inputs: z.array(z.string()),
          actions: z.array(z.object({ label: z.string(), prompt: z.string() })),
          trigger: z.enum(["manual", "webhook", "app"]),
          version: z.number(),
          createdAt: z.date(),
        }),
        otherVersions: z.array(
          z.object({
            id: z.string(),
            version: z.number(),
          }),
        ),
      }),
      401: z.undefined(),
      404: z.object({
        message: z.string(),
      }),
    },
    pathParams: z.object({
      clusterId: z.string(),
      templateId: z.string(),
    }),
  },
  getWorkflowDetail: {
    method: "GET",
    path: "/clusters/:clusterId/workflows/:workflowId",
    headers: z.object({
      authorization: z.string(),
    }),
    responses: {
      200: z.object({
        workflow: z.object({
          id: z.string(),
          jobHandle: z.string().nullable(),
          userId: z.string().nullable(),
          actions: z
            .array(
              z.object({
                label: z.string(),
                prompt: z.string(),
              }),
            )
            .nullable(),
          status: z.string().nullable(),
          failureReason: z.string().nullable(),
          test: z.boolean(),
          feedbackComment: z.string().nullable(),
          feedbackScore: z.number().nullable(),
        }),
        inputRequests: z.array(
          z.object({
            id: z.string(),
            type: z.string(),
            requestArgs: z.string().optional().nullable(),
            resolvedAt: z.date().nullable().optional(),
            createdAt: z.date(),
            service: z.string().nullable().optional(),
            function: z.string().nullable().optional(),
            description: z.string().nullable().optional(),
            presentedOptions: z.array(z.string()).nullable().optional(),
          }),
        ),
        jobs: z.array(
          z.object({
            id: z.string(),
            targetFn: z.string(),
            status: z.string(),
            service: z.string(),
            resultType: z.string().nullable(),
            createdAt: z.date(),
          }),
        ),
      }),
      401: z.undefined(),
    },
  },
  submitWorkflowFeedback: {
    method: "POST",
    path: "/clusters/:clusterId/workflows/:workflowId/feedback",
    headers: z.object({
      authorization: z.string(),
    }),
    body: z.object({
      comment: z.string().describe("Feedback comment"),
      score: z.number().describe("Score between 1 and 10").min(1).max(10),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
      404: z.undefined(),
    },
    pathParams: z.object({
      workflowId: z.string(),
      clusterId: z.string(),
    }),
  },
  resolveInputRequest: {
    method: "POST",
    path: "/clusters/:clusterId/workflows/:workflowId/input-requests/:inputRequestId",
    headers: z.object({
      authorization: z.string().optional(),
    }),
    body: z.object({
      input: z.string(),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
      404: z.undefined(),
    },
    pathParams: z.object({
      workflowId: z.string(),
      inputRequestId: z.string(),
      clusterId: z.string(),
    }),
  },
  getInputRequest: {
    method: "GET",
    path: "/clusters/:clusterId/workflows/:workflowId/input-requests/:inputRequestId",
    headers: z.object({
      authorization: z.string().optional(),
    }),
    pathParams: z.object({
      clusterId: z.string(),
      workflowId: z.string(),
      inputRequestId: z.string(),
    }),
    responses: {
      200: z.object({
        id: z.string(),
        workflowId: z.string(),
        clusterId: z.string(),
        resolvedAt: z.date().nullable(),
        createdAt: z.date(),
        requestArgs: z.string().nullable(),
        service: z.string().nullable(),
        function: z.string().nullable(),
        description: z.string().nullable(),
        type: z.string(),
        options: z.array(z.string()).optional(),
      }),
      401: z.undefined(),
      404: z.undefined(),
    },
  },
  oas: {
    method: "GET",
    path: "/public/oas.json",
    responses: {
      200: z.unknown(),
    },
  },
  pingCluster: {
    method: "POST",
    path: "/ping-cluster",
    headers: z.object({
      authorization: z.string(),
      "x-machine-id": z.string(),
      "x-machine-sdk-version": z.string(),
      "x-machine-sdk-language": z.string(),
    }),
    body: z.object({
      services: z.array(z.string()),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
  },
  pingClusterV2: {
    method: "POST",
    path: "/ping-cluster-v2",
    headers: z.object({
      authorization: z.string(),
      "x-machine-id": z.string(),
      "x-machine-sdk-version": z.string(),
      "x-machine-sdk-language": z.string(),
    }),
    body: z.object({
      services: z.array(z.string()),
    }),
    responses: {
      200: z.object({
        outdated: z.boolean(),
      }),
      401: z.undefined(),
    },
  },
  editHumanMessage: {
    method: "PUT",
    path: "/clusters/:clusterId/workflows/:workflowId/messages/:messageId",
    headers: z.object({ authorization: z.string() }),
    body: z.object({ message: z.string() }),
    responses: {
      200: z.object({
        data: genericMessageDataSchema,
        id: z.string(),
      }),
      404: z.object({ message: z.string() }),
      401: z.undefined(),
    },
    pathParams: z.object({
      clusterId: z.string(),
      workflowId: z.string(),
      messageId: z.string(),
    }),
  },
  triggerTemplateWithContext: {
    method: "POST",
    path: "/clusters/:clusterId/templates/:templateId/execute-with-context",
    headers: z.object({ authorization: z.string() }),
    body: z.object({
      url: z.string(),
    }),
    pathParams: z.object({
      clusterId: z.string(),
      templateId: z.string(),
    }),
    responses: {
      200: z.object({
        url: z.string(),
        workflowId: z.string(),
      }),
    },
  },
  storeServiceMetadata: {
    method: "PUT",
    path: "/clusters/:clusterId/services/:service/keys/:key",
    headers: z.object({ authorization: z.string() }),
    body: z.object({
      value: z.string(),
    }),
    pathParams: z.object({
      clusterId: z.string(),
      service: z.string(),
      key: z.string(),
    }),
    responses: {
      204: z.undefined(),
      401: z.undefined(),
    },
  },
  interceptWebhook: {
    method: "POST",
    path: "/clusters/:clusterId/services/:service/events",
    body: z.unknown(),
    pathParams: z.object({
      clusterId: z.string(),
      service: z.string(),
    }),
    responses: {
      200: z.undefined(),
    },
  },
  getClusterExport: {
    method: "GET",
    path: "/clusters/:clusterId/export",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({ clusterId: z.string() }),
    responses: {
      200: z.object({
        data: z.string(),
      }),
    },
  },
  consumeClusterExport: {
    method: "POST",
    path: "/clusters/:clusterId/import",
    headers: z.object({ authorization: z.string() }),
    body: z.object({ data: z.string() }),
    pathParams: z.object({ clusterId: z.string() }),
    responses: {
      200: z.object({ message: z.string() }),
      400: z.undefined(),
    },
  },
  getActivityByWorkflowIdForUserAttentionLevel: {
    method: "GET",
    path: "/clusters/:clusterId/workflows/:workflowId/activity",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
      workflowId: z.string(),
    }),
    query: z.object({
      userAttentionLevel: z.enum(["debug", "info", "warning", "error"]),
    }),
    responses: {
      200: z.array(
        z.object({
          type: z.string(),
          machineId: z.string().nullable(),
          service: z.string().nullable(),
          createdAt: z.string(),
          jobId: z.string().nullable(),
          targetFn: z.string().nullable(),
          resultType: z.string().nullable(),
          status: z.string().nullable(),
          workflowId: z.string().nullable(),
          meta: z.any().nullable(),
          id: z.string(),
        }),
      ),
    },
  },
  getJobDetail: {
    method: "GET",
    path: "/clusters/:clusterId/jobs/:jobId",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
      jobId: z.string(),
    }),
    responses: {
      200: z.object({
        id: z.string(),
        status: z.string(),
        targetFn: z.string(),
        service: z.string(),
        executingMachineId: z.string().nullable(),
        targetArgs: z.string(),
        result: z.string().nullable(),
        resultType: z.string().nullable(),
        createdAt: z.date(),
      }),
    },
  },
  getJobReferences: {
    method: "GET",
    path: "/clusters/:clusterId/workflows/:workflowId/job-references",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
      workflowId: z.string(),
    }),
    query: z.object({
      token: z.string(),
      before: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          result: z.string().nullable(),
          createdAt: z.date(),
          status: z.string(),
          targetFn: z.string(),
          service: z.string(),
          executingMachineId: z.string().nullable(),
        }),
      ),
    },
  },
  createApiKey: {
    method: "POST",
    path: "/clusters/:clusterId/api-keys",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
    }),
    body: z.object({
      name: z.string(),
      type: z.enum(["cluster_manage", "cluster_consume", "cluster_machine"]),
    }),
    responses: {
      200: z.object({
        id: z.string(),
        key: z.string(),
      }),
    },
  },
  listApiKeys: {
    method: "GET",
    path: "/clusters/:clusterId/api-keys",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
    }),
    responses: {
      200: z.array(
        z.object({
          id: z.string(),
          name: z.string(),
          type: z.enum([
            "cluster_manage",
            "cluster_consume",
            "cluster_machine",
          ]),
          createdAt: z.date(),
          createdBy: z.string(),
          revokedAt: z.date().nullable(),
        }),
      ),
    },
  },
  revokeApiKey: {
    method: "DELETE",
    path: "/clusters/:clusterId/api-keys/:keyId",
    headers: z.object({ authorization: z.string() }),
    pathParams: z.object({
      clusterId: z.string(),
      keyId: z.string(),
    }),
    body: z.undefined(),
    responses: {
      204: z.undefined(),
    },
  },
} as const;

export const contract = c.router(definition);
