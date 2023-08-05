package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

func OpenAPICtx(c *gin.Context) openapi.OperationContext {
	if oc, ok := c.Get("openapiContext"); ok {
		if oc, ok := oc.(openapi.OperationContext); ok {
			return oc
		}
	}

	return nil
}

func OpenAPICollect(refl openapi.Reflector, routes gin.RoutesInfo) error {
	pathReplace := regexp.MustCompile(`:([^/.]+)`)

	for _, r := range routes {
		fmt.Println(r.Method, r.Path, r.Handler)

		pathPattern := pathReplace.ReplaceAllString(r.Path, "{$1}")

		oc, err := refl.NewOperationContext(r.Method, pathPattern)
		if err != nil {
			return err
		}

		c := &gin.Context{}
		c.Set("openapiContext", oc)

		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
				}
			}()
			r.HandlerFunc(c)
		}()

		if len(oc.Request()) == 0 && len(oc.Response()) == 0 {
			_, _, pathItems, err := openapi.SanitizeMethodPath(r.Method, pathPattern)
			if err != nil {
				return err
			}

			if len(pathItems) > 0 {
				if o3, ok := oc.(openapi3.OperationExposer); ok {
					op := o3.Operation()

					for _, p := range pathItems {
						param := openapi3.ParameterOrRef{}
						param.WithParameter(openapi3.Parameter{
							Name: p,
							In:   openapi3.ParameterInPath,
						})

						op.Parameters = append(op.Parameters, param)
					}
				}
			}

			oc.SetDescription("Information about this operation was obtained using only HTTP method and path pattern. " +
				"It may be incomplete and/or inaccurate.")
			oc.SetTags("Incomplete")
			oc.AddRespStructure(nil, func(cu *openapi.ContentUnit) {
				cu.ContentType = "text/html"
			})
		}

		if err := refl.AddOperation(oc); err != nil {
			return err
		}

	}

	return nil
}

// album represents data about a record album.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

// albums slice to seed record album data.
var albums = []album{
	{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
	{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
	{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

func main() {
	router := gin.Default()
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)

	refl := openapi3.NewReflector()
	refl.SpecSchema().SetTitle("Albums API")
	refl.SpecSchema().SetVersion("v1.2.3")
	refl.SpecSchema().SetDescription("This services keeps track of albums.")

	if err := OpenAPICollect(refl, router.Routes()); err != nil {
		fmt.Println(err.Error())
	}

	y, _ := refl.Spec.MarshalYAML()

	os.WriteFile("openapi.yaml", y, 0600)
	fmt.Println(string(y))

	router.Run("localhost:8080")
}

// getAlbums responds with the list of all albums as JSON.
func getAlbums(c *gin.Context) {
	if oc := OpenAPICtx(c); oc != nil {
		oc.SetSummary("List albums")
		oc.SetTags("Albums")
		oc.AddRespStructure(albums)
	}

	c.JSON(http.StatusOK, albums)
}

// postAlbums adds an album from JSON received in the request body.
func postAlbums(c *gin.Context) {
	var newAlbum album

	if oc := OpenAPICtx(c); oc != nil {
		oc.SetSummary("Create an album")
		oc.SetTags("Albums")
		oc.AddReqStructure(newAlbum)
		oc.AddRespStructure(newAlbum)
	}

	// Call BindJSON to bind the received JSON to
	// newAlbum.
	if err := c.BindJSON(&newAlbum); err != nil {
		return
	}

	// Add the new album to the slice.
	albums = append(albums, newAlbum)
	c.JSON(http.StatusCreated, newAlbum)
}

// getAlbumByID locates the album whose ID value matches the id
// parameter sent by the client, then returns that album as a response.
func getAlbumByID(c *gin.Context) {
	if oc := OpenAPICtx(c); oc != nil {
		oc.SetSummary("Get album")
		oc.SetTags("Albums")
		oc.AddReqStructure(struct {
			ID string `path:"id"`
		}{})
		oc.AddRespStructure(album{})
		oc.AddRespStructure(struct {
			Message string `json:"message"`
		}{}, openapi.WithHTTPStatus(http.StatusNotFound))
	}

	id := c.Param("id")

	// Loop through the list of albums, looking for
	// an album whose ID value matches the parameter.
	for _, a := range albums {
		if a.ID == id {
			c.JSON(http.StatusOK, a)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "album not found"})
}
