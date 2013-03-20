class Orvos::V1::SchemaController < ApplicationController
  skip_before_filter :find_object_by_uuid
  skip_before_filter :login_required

  def show
    classes = Rails.cache.fetch 'orvos_v1_schema' do
      Rails.application.eager_load!
      classes = {}
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |k|
        classes[k] = k.columns.collect do |col|
          if k.serialized_attributes.has_key? col.name
            { name: col.name,
              type: k.serialized_attributes[col.name].object_class.to_s }
          else
            { name: col.name,
              type: col.type }
          end
        end
      end
      classes
    end
    render json: classes
  end

  def discovery_rest_description
    discovery = Rails.cache.fetch 'orvos_v1_rest_discovery' do
      Rails.application.eager_load!
      discovery = {
        kind: "discovery#restDescription",
        discoveryVersion: "v1",
        id: "orvos:v1",
        name: "orvos",
        version: "v1",
        revision: "20130226",
        title: "Orvos API",
        description: "The API to interact with Orvos.",
        documentationLink: "https://redmine.clinicalfuture.com/projects/orvos/",
        protocol: "rest",
        baseUrl: root_url + "/orvos/v1/",
        basePath: "/orvos/v1/",
        rootUrl: root_url,
        servicePath: "orvos/v1/",
        batchPath: "batch",
        parameters: {
          alt: {
            type: "string",
            description: "Data format for the response.",
            default: "json",
            enum: [
                   "json"
                  ],
            enumDescriptions: [
                               "Responses with Content-Type of application/json"
                              ],
            location: "query"
          },
          fields: {
            type: "string",
            description: "Selector specifying which fields to include in a partial response.",
            location: "query"
          },
          key: {
            type: "string",
            description: "API key. Your API key identifies your project and provides you with API access, quota, and reports. Required unless you provide an OAuth 2.0 token.",
            location: "query"
          },
          oauth_token: {
            type: "string",
            description: "OAuth 2.0 token for the current user.",
            location: "query"
          }
        },
        auth: {
          oauth2: {
            scopes: {
              "https://api.clinicalfuture.com/auth/orvos" => {
                description: "View and manage objects"
              },
              "https://api.clinicalfuture.com/auth/orvos.readonly" => {
                description: "View objects"
              }
            }
          }
        },
        schemas: {},
        resources: {}
      }
      
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |k|
        object_properties = {}
        k.columns.
          select { |col| col.name != 'id' }.
          collect do |col|
          if k.serialized_attributes.has_key? col.name
            object_properties[col.name] = {
              type: k.serialized_attributes[col.name].object_class.to_s
            }
          else
            object_properties[col.name] = {
              type: col.type
            }
          end
        end
        discovery[:schemas][k.to_s + 'List'] = {
          id: k.to_s,
          description: k.to_s,
          type: "object",
          properties: {
            kind: {
              type: "string",
              description: "Object type. Always orvos##{k.to_s.camelcase(:lower)}List.",
              default: "orvos##{k.to_s.camelcase(:lower)}List"
            },
            etag: {
              type: "string",
              description: "List version."
            },
            items: {
              type: "array",
              description: "The list of #{k.to_s.pluralize}."
            },
            next_link: {
              type: "string",
              description: "A link to the next page of #{k.to_s.pluralize}."
            },
            next_page_token: {
              type: "string",
              description: "The page token for the next page of #{k.to_s.pluralize}."
            },
            selfLink: {
              type: "string",
              description: "A link back to this list."
            }
          }
        }
        discovery[:schemas][k.to_s] = {
          id: k.to_s,
          description: k.to_s,
          type: "object",
          properties: {
            uuid: {
              type: "string",
              description: "Object ID."
            },
            etag: {
              type: "string",
              description: "Object version."
            }
          }.merge(object_properties)
        }
        discovery[:resources][k.to_s.underscore.pluralize] = {
          methods: {
            get: {
              id: "orvos.#{k.to_s.underscore.pluralize}.get",
              path: "#{k.to_s.underscore.pluralize}/{uuid}",
              httpMethod: "GET",
              description: "Gets a #{k.to_s}'s metadata by ID.",
              parameters: {
                uuid: {
                  type: "string",
                  description: "The ID for the #{k.to_s} in question.",
                  required: true,
                  location: "path"
                }
              },
              parameterOrder: [
                               "uuid"
                              ],
              response: {
                "$ref" => k.to_s
              },
              scopes: [
                       "https://api.clinicalfuture.com/auth/orvos",
                       "https://api.clinicalfuture.com/auth/orvos.readonly"
                      ]
            },
            list: {
              id: "orvos.#{k.to_s.underscore.pluralize}.list",
              path: k.to_s.underscore.pluralize,
              httpMethod: "GET",
              description: "List #{k.to_s.underscore.pluralize}.",
              parameters: {
                limit: {
                  type: "integer",
                  description: "Maximum number of #{k.to_s.underscore.pluralize} to return.",
                  default: "100",
                  format: "int32",
                  minimum: "0",
                  location: "query"
                },
                pageToken: {
                  type: "string",
                  description: "Page token.",
                  location: "query"
                },
                q: {
                  type: "string",
                  description: "Query string for searching #{k.to_s.underscore.pluralize}.",
                  location: "query"
                }
              },
              response: {
                "$ref" => "#{k.to_s.pluralize}List"
              },
              scopes: [
                       "https://api.clinicalfuture.com/auth/orvos",
                       "https://api.clinicalfuture.com/auth/orvos.readonly"
                      ]
            },
            create: {
              id: "orvos.#{k.to_s.underscore.pluralize}.create",
              path: "#{k.to_s.underscore.pluralize}",
              httpMethod: "POST",
              description: "Create a new #{k.to_s}.",
              parameters: {
                k.to_s.underscore => {
                  type: "object",
                  required: true,
                  location: "query",
                  properties: object_properties
                }
              },
              request: {
                "$ref" => k.to_s
              },
              response: {
                "$ref" => k.to_s
              },
              scopes: [
                       "https://api.clinicalfuture.com/auth/orvos"
                      ]
            },
            update: {
              id: "orvos.#{k.to_s.underscore.pluralize}.update",
              path: "#{k.to_s.underscore.pluralize}/{uuid}",
              httpMethod: "PUT",
              description: "Update attributes of an existing #{k.to_s}.",
              parameters: {
                uuid: {
                  type: "string",
                  description: "The ID for the #{k.to_s} in question.",
                  required: true,
                  location: "path"
                },
                k.to_s.underscore => {
                  type: "object",
                  required: true,
                  location: "query",
                  properties: object_properties
                }
              },
              request: {
                "$ref" => k.to_s
              },
              response: {
                "$ref" => k.to_s
              },
              scopes: [
                       "https://api.clinicalfuture.com/auth/orvos"
                      ]
            }
          }
        }
      end
      discovery
    end
    render json: discovery
  end
end
