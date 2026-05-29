MAIN    = main
LATEX   = pdflatex
FLAGS   = -interaction=nonstopmode -halt-on-error

.PHONY: pdf clean distclean

pdf:
	$(LATEX) $(FLAGS) $(MAIN).tex
	$(LATEX) $(FLAGS) $(MAIN).tex
	@echo "Done: $(MAIN).pdf"

clean:
	rm -f $(MAIN).{aux,log,toc,out,lof,lot}
	rm -f chapters/*.aux

distclean: clean
	rm -f $(MAIN).pdf
