package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "Amin Muborakkadamov",
            "url": "https://github.com/AminVamar"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/v1/admin/forms": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "формы"
                ],
                "summary": "Список созданных форм",
                "parameters": [
                    {
                        "type": "string",
                        "description": "open или closed",
                        "name": "status",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "лимит (до 200)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "смещение",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/http.ListResponse"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": {
                                                "$ref": "#/definitions/domain.FormSession"
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            },
            "post": {
                "description": "Создаёт Google-форму по тесту. Стажёрам отдавать respondent_url.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "формы"
                ],
                "summary": "Создать форму",
                "parameters": [
                    {
                        "description": "тест и время на прохождение",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/http.CreateFormRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.FormSession"
                        }
                    },
                    "400": {
                        "description": "неверные данные",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "тест не найден",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "503": {
                        "description": "сервис недоступен",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/admin/forms/{id}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "формы"
                ],
                "summary": "Форма по идентификатору",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.FormSession"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/admin/forms/{id}/close": {
            "post": {
                "description": "Форма перестаёт принимать ответы.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "формы"
                ],
                "summary": "Закрыть форму",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.FormSession"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "503": {
                        "description": "Apps Script недоступен",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/admin/grading/attempts/{id}": {
            "get": {
                "description": "Ответы стажёра с правильными ответами.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "проверка"
                ],
                "summary": "Попытка для проверки",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "boolean",
                        "description": "только непроверенные",
                        "name": "only_pending",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.AttemptResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            },
            "patch": {
                "description": "Балл: 0, 1 или null.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "проверка"
                ],
                "summary": "Выставить баллы",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "баллы",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/http.GradeRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.GradeResponse"
                        }
                    },
                    "400": {
                        "description": "неверный балл",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/admin/grading/pending": {
            "get": {
                "description": "Попытки с непроверенными ответами.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "проверка"
                ],
                "summary": "Очередь на ручную проверку",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "фильтр по филиалу",
                        "name": "branch_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "фильтр по тесту",
                        "name": "test_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "лимит (до 200)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "смещение",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/http.ListResponse"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": {
                                                "$ref": "#/definitions/domain.Attempt"
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            }
        },
        "/api/v1/attempts/{id}": {
            "get": {
                "description": "Все ответы и итоговый балл.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "попытки"
                ],
                "summary": "Попытка целиком",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.AttemptResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/attempts/{id}/rescore": {
            "post": {
                "description": "Ручные баллы не меняются.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "попытки"
                ],
                "summary": "Пересчёт автоматических баллов",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.RescoreResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "503": {
                        "description": "источник тестов недоступен",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/branches": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "филиалы"
                ],
                "summary": "Филиалы",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.BranchesResponse"
                        }
                    }
                }
            },
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "филиалы"
                ],
                "summary": "Добавить филиал",
                "parameters": [
                    {
                        "description": "филиал",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/http.CreateBranchRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/domain.Branch"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/interns": {
            "get": {
                "description": "Поиск по ФИО или телефону.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "стажёры"
                ],
                "summary": "Стажёры",
                "parameters": [
                    {
                        "type": "string",
                        "description": "часть ФИО или телефона",
                        "name": "search",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "фильтр по филиалу",
                        "name": "branch_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "лимит (до 200)",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "смещение",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/http.ListResponse"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "items": {
                                            "type": "array",
                                            "items": {
                                                "$ref": "#/definitions/domain.InternListItem"
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    }
                }
            }
        },
        "/api/v1/interns/{id}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "стажёры"
                ],
                "summary": "Стажёр и его попытки",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "фильтр по тесту",
                        "name": "test_id",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "graded или needs_grading",
                        "name": "status",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "лимит",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "смещение",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.InternCardResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/tests": {
            "get": {
                "description": "Список тестов из внешнего источника.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "тесты"
                ],
                "summary": "Все тесты из источника",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.TestsResponse"
                        }
                    },
                    "503": {
                        "description": "источник тестов недоступен",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/tests/{id}": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "тесты"
                ],
                "summary": "Тест по идентификатору",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/domain.Test"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "503": {
                        "description": "источник тестов недоступен",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/webhook/ping": {
            "post": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "вебхук"
                ],
                "summary": "Проверка секрета вебхука",
                "parameters": [
                    {
                        "type": "string",
                        "description": "общий секрет",
                        "name": "X-Gform-Secret",
                        "in": "header",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.PingResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/api/v1/webhook/submission": {
            "post": {
                "description": "Вызывается из Apps Script, секрет в X-Gform-Secret.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "вебхук"
                ],
                "summary": "Приём ответа стажёра",
                "parameters": [
                    {
                        "type": "string",
                        "description": "общий секрет",
                        "name": "X-Gform-Secret",
                        "in": "header",
                        "required": true
                    },
                    {
                        "description": "ответ стажёра",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/domain.Submission"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.SubmissionResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "неверный секрет",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "форма не найдена",
                        "schema": {
                            "$ref": "#/definitions/http.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/health": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "служебное"
                ],
                "summary": "Health check",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/http.HealthResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "domain.Answer": {
            "type": "object",
            "properties": {
                "answer_text": {
                    "type": "string"
                },
                "attempt_id": {
                    "type": "integer"
                },
                "correct_answers": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "graded_at": {
                    "type": "string"
                },
                "graded_by": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "item_id": {
                    "type": "string"
                },
                "position": {
                    "type": "integer"
                },
                "question_id": {
                    "type": "integer"
                },
                "question_text": {
                    "type": "string"
                },
                "question_type": {
                    "type": "integer"
                },
                "score": {
                    "type": "integer"
                }
            }
        },
        "domain.Attempt": {
            "type": "object",
            "properties": {
                "auto_submitted": {
                    "type": "boolean"
                },
                "branch_code": {
                    "type": "string"
                },
                "branch_id": {
                    "type": "integer"
                },
                "branch_name": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "form_id": {
                    "type": "string"
                },
                "form_session_id": {
                    "type": "integer"
                },
                "id": {
                    "type": "integer"
                },
                "intern_id": {
                    "type": "integer"
                },
                "intern_name": {
                    "type": "string"
                },
                "intern_phone": {
                    "type": "string"
                },
                "needs_grading": {
                    "type": "boolean"
                },
                "pending_count": {
                    "type": "integer"
                },
                "response_id": {
                    "type": "string"
                },
                "scored": {
                    "type": "integer"
                },
                "submitted_at": {
                    "type": "string"
                },
                "test_id": {
                    "type": "integer"
                },
                "test_title": {
                    "type": "string"
                },
                "total_questions": {
                    "type": "integer"
                }
            }
        },
        "domain.Branch": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "domain.FormSession": {
            "type": "object",
            "properties": {
                "closed_at": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "duration_minutes": {
                    "type": "integer"
                },
                "edit_url": {
                    "type": "string"
                },
                "form_id": {
                    "type": "string"
                },
                "form_url": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "item_map": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "integer",
                        "format": "int64"
                    }
                },
                "question_count": {
                    "type": "integer"
                },
                "respondent_url": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "test_id": {
                    "type": "integer"
                },
                "test_title": {
                    "type": "string"
                }
            }
        },
        "domain.GradeInput": {
            "type": "object",
            "properties": {
                "answer_id": {
                    "type": "integer"
                },
                "score": {
                    "type": "integer"
                }
            }
        },
        "domain.Intern": {
            "type": "object",
            "properties": {
                "branch_code": {
                    "type": "string"
                },
                "branch_id": {
                    "type": "integer"
                },
                "branch_name": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "first_name": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "last_name": {
                    "type": "string"
                },
                "middle_name": {
                    "type": "string"
                },
                "phone": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.InternListItem": {
            "type": "object",
            "properties": {
                "attempts_count": {
                    "type": "integer"
                },
                "branch_code": {
                    "type": "string"
                },
                "branch_id": {
                    "type": "integer"
                },
                "branch_name": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "first_name": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "last_name": {
                    "type": "string"
                },
                "middle_name": {
                    "type": "string"
                },
                "needs_grading": {
                    "type": "boolean"
                },
                "phone": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "domain.Option": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer"
                },
                "is_correct": {
                    "type": "boolean"
                },
                "question_id": {
                    "type": "integer"
                },
                "text": {
                    "type": "string"
                }
            }
        },
        "domain.Question": {
            "type": "object",
            "properties": {
                "content": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "options": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Option"
                    }
                },
                "type": {
                    "type": "integer"
                }
            }
        },
        "domain.Submission": {
            "type": "object",
            "properties": {
                "answers": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.SubmittedAnswer"
                    }
                },
                "auto_submitted": {
                    "type": "boolean"
                },
                "branch_code": {
                    "type": "string"
                },
                "branch_name": {
                    "type": "string"
                },
                "first_name": {
                    "type": "string"
                },
                "form_id": {
                    "type": "string"
                },
                "last_name": {
                    "type": "string"
                },
                "middle_name": {
                    "type": "string"
                },
                "phone": {
                    "type": "string"
                },
                "response_id": {
                    "type": "string"
                },
                "submitted_at": {
                    "type": "string"
                }
            }
        },
        "domain.SubmittedAnswer": {
            "type": "object",
            "properties": {
                "answer": {
                    "type": "string"
                },
                "item_id": {
                    "type": "string"
                },
                "question_id": {
                    "type": "integer"
                }
            }
        },
        "domain.Test": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "questions": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Question"
                    }
                },
                "test_type": {
                    "type": "integer"
                },
                "title": {
                    "type": "string"
                }
            }
        },
        "http.AttemptResponse": {
            "type": "object",
            "properties": {
                "answers": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Answer"
                    }
                },
                "attempt": {
                    "$ref": "#/definitions/domain.Attempt"
                },
                "score": {
                    "type": "integer"
                },
                "source_available": {
                    "type": "boolean"
                },
                "status": {
                    "type": "string"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.BranchesResponse": {
            "type": "object",
            "properties": {
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Branch"
                    }
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.CreateBranchRequest": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string",
                    "example": "6900"
                },
                "name": {
                    "type": "string",
                    "example": "ЦБО Рудаки"
                }
            }
        },
        "http.CreateFormRequest": {
            "type": "object",
            "properties": {
                "duration_minutes": {
                    "type": "integer",
                    "example": 30
                },
                "test_id": {
                    "type": "integer",
                    "example": 35
                }
            }
        },
        "http.ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                }
            }
        },
        "http.GradeRequest": {
            "type": "object",
            "properties": {
                "grades": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.GradeInput"
                    }
                }
            }
        },
        "http.GradeResponse": {
            "type": "object",
            "properties": {
                "attempt": {
                    "$ref": "#/definitions/http.AttemptResponse"
                },
                "attempt_id": {
                    "type": "integer"
                },
                "changed": {
                    "type": "integer"
                },
                "score": {
                    "type": "integer"
                },
                "status": {
                    "type": "string",
                    "example": "graded"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.HealthResponse": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "ok"
                }
            }
        },
        "http.InternCardResponse": {
            "type": "object",
            "properties": {
                "attempts": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Attempt"
                    }
                },
                "attempts_total": {
                    "type": "integer"
                },
                "intern": {
                    "$ref": "#/definitions/domain.Intern"
                }
            }
        },
        "http.ListResponse": {
            "type": "object",
            "properties": {
                "items": {},
                "limit": {
                    "type": "integer"
                },
                "offset": {
                    "type": "integer"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.PingResponse": {
            "type": "object",
            "properties": {
                "ok": {
                    "type": "boolean",
                    "example": true
                }
            }
        },
        "http.RescoreResponse": {
            "type": "object",
            "properties": {
                "attempt_id": {
                    "type": "integer"
                },
                "changed": {
                    "type": "integer"
                },
                "score": {
                    "type": "integer"
                },
                "status": {
                    "type": "string"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.SubmissionResponse": {
            "type": "object",
            "properties": {
                "attempt_id": {
                    "type": "integer"
                },
                "duplicate": {
                    "type": "boolean"
                },
                "intern_id": {
                    "type": "integer"
                },
                "needs_grading": {
                    "type": "boolean"
                },
                "score": {
                    "type": "integer"
                },
                "scored": {
                    "type": "boolean"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "http.TestsResponse": {
            "type": "object",
            "properties": {
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/domain.Test"
                    }
                },
                "total": {
                    "type": "integer"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Activ GForms API",
	Description:      "API для тестирования стажёров Активбанка через Google Forms.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
