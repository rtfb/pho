
GOFILES=\
	*.go

BUILDDIR=build
JSDIR=${BUILDDIR}/static/js
CSSDIR=${BUILDDIR}/static/css

TMPL_FILES = $(notdir $(wildcard tmpl/*.html))
TARGETS = \
	$(addprefix ${BUILDDIR}/tmpl/, $(TMPL_FILES)) \
	${JSDIR}/lightgallery.min.js \
	${JSDIR}/lg-thumbnail.js \
	${JSDIR}/jquery.mousewheel.min.js \
	${JSDIR}/picturefill.min.js \
	${JSDIR}/jquery.min.js \
	${JSDIR}/jquery.uploadfile.min.js \
	${CSSDIR}/uploadfile.css \
	${CSSDIR}/lightgallery.css

all: ${BUILDDIR}/pho

bootstrap: schema
	glide install
	bower install

schema:
	sql-migrate up -config=./db/dbconfig.yaml

run: all
	./${BUILDDIR}/pho

${BUILDDIR}/pho: $(GOFILES) $(TARGETS)
	go build -o ${BUILDDIR}/pho $(GOFILES)

${BUILDDIR}/tmpl/%.html: tmpl/%.html
	@mkdir -p ${BUILDDIR}/tmpl
	cp $< $@

${CSSDIR}/lightgallery.css: bower_components/lightgallery/dist/css/lightgallery.css
	@mkdir -p ${CSSDIR}
	cp $< $@

${CSSDIR}/uploadfile.css: bower_components/jquery-uploadfile/css/uploadfile.css
	@mkdir -p ${CSSDIR}
	cp $< $@

${JSDIR}/lightgallery.min.js: bower_components/lightgallery/dist/js/lightgallery.min.js
	@mkdir -p ${JSDIR}
	cp $< $@

${JSDIR}/lg-thumbnail.js: bower_components/lightgallery/demo/js/lg-thumbnail.js
	@mkdir -p ${JSDIR}
	cp $< $@

${JSDIR}/jquery.mousewheel.min.js: bower_components/lightgallery/lib/jquery.mousewheel.min.js
	@mkdir -p ${JSDIR}
	cp $< $@

${JSDIR}/picturefill.min.js: bower_components/lightgallery/lib/picturefill.min.js
	@mkdir -p ${JSDIR}
	cp $< $@

${JSDIR}/jquery.min.js: bower_components/jquery/dist/jquery.min.js
	@mkdir -p ${JSDIR}
	cp $< $@

${JSDIR}/jquery.uploadfile.min.js: bower_components/jquery-uploadfile/js/jquery.uploadfile.min.js
	@mkdir -p ${JSDIR}
	cp $< $@
