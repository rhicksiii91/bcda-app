package cclf

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

func CreateACO(name, cmsID string) (string, error) {
	if name == "" {
		return "", errors.New("ACO name (--name) must be provided")
	}

	var cmsIDPt *string
	if cmsID != "" {
		acoIDFmt := regexp.MustCompile(`^A\d{4}$`)
		if !acoIDFmt.MatchString(cmsID) {
			return "", errors.New("ACO CMS ID (--cms-id) is invalid")
		}
		cmsIDPt = &cmsID
	}

	acoUUID, err := models.CreateACO(name, cmsIDPt)
	if err != nil {
		return "", err
	}

	return acoUUID.String(), nil
}

type cclfFileMetadata struct {
	name         string
	env          string
	acoID        string
	cclfNum      int
	perfYear     int
	timestamp    time.Time
	filePath     string
	imported     bool
	deliveryDate time.Time
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

const deleteThresholdHr = 8

func importCCLF0(fileMetadata *cclfFileMetadata) (map[string]cclfFileValidator, error) {
	if fileMetadata == nil {
		fmt.Println("File CCLF0 not found.")
		err := errors.New("file CCLF0 not found")
		log.Error(err)
		return nil, err
	}

	fmt.Printf("Importing CCLF0 file %s...\n", fileMetadata.name)
	log.Infof("Importing CCLF0 file %s...", fileMetadata.name)

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read CCLF0 archive %s.\n", fileMetadata.name)
		err := errors.Wrapf(err, "could not read CCLF0 archive %s", fileMetadata.name)
		log.Error(err)
		return nil, err
	}
	defer r.Close()

	const (
		fileNumStart, fileNumEnd           = 0, 7
		totalRecordStart, totalRecordEnd   = 52, 63
		recordLengthStart, recordLengthEnd = 64, 69
	)

	var validator map[string]cclfFileValidator
	for i, f := range r.File {
		fmt.Printf("Reading file #%d from archive %s.\n", i, fileMetadata.name)
		log.Infof("Reading file #%d from archive %s", i, fileMetadata.name)
		rc, err := f.Open()
		if err != nil {
			fmt.Printf("Could not read file %s in CCLF0 archive %s.\n", f.Name, fileMetadata.name)
			err = errors.Wrapf(err, "could not read file %s in CCLF0 archive %s", f.Name, fileMetadata.name)
			log.Error(err)
			return nil, err
		}
		defer rc.Close()
		sc := bufio.NewScanner(rc)
		for sc.Scan() {
			b := sc.Bytes()
			if len(bytes.TrimSpace(b)) > 0 {
				filetype := string(bytes.TrimSpace(b[fileNumStart:fileNumEnd]))

				if filetype == "CCLF8" || filetype == "CCLF9" {
					if validator == nil {
						validator = make(map[string]cclfFileValidator)
					}
					count, err := strconv.Atoi(string(bytes.TrimSpace(b[totalRecordStart:totalRecordEnd])))
					if err != nil {
						fmt.Printf("Failed to parse %s record count from CCLF0 file.\n", filetype)
						err = errors.Wrapf(err, "failed to parse %s record count from CCLF0 file", filetype)
						log.Error(err)
						return nil, err
					}
					length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
					if err != nil {
						fmt.Printf("Failed to parse %s record length from CCLF0 file.\n", filetype)
						err = errors.Wrapf(err, "failed to parse %s record length from CCLF0 file", filetype)
						log.Error(err)
						return nil, err
					}
					validator[filetype] = cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
				}
			}
		}
	}

	if _, ok := validator["CCLF8"]; !ok {
		fmt.Printf("Failed to parse CCLF8 from CCLF0 file: %#v.\n", fileMetadata)
		err := fmt.Errorf("failed to parse CCLF8 from CCLF0 file: %#v", fileMetadata)
		log.Error(err)
		return nil, err
	}
	if _, ok := validator["CCLF9"]; !ok {
		fmt.Printf("Failed to parse CCLF9 from CCLF0 file: %#v.\n", fileMetadata)
		err := fmt.Errorf("failed to parse CCLF9 from CCLF0 file: %#v", fileMetadata)
		log.Error(err)
		return nil, err
	}
	fmt.Printf("Successfully imported CCLF0 file %s.\n", fileMetadata.name)
	log.Infof("Successfully imported CCLF0 file %s.", fileMetadata.name)

	return validator, nil
}

func importCCLF8(fileMetadata *cclfFileMetadata) error {
	err := importCCLF(fileMetadata, func(fileID uint, b []byte, db *gorm.DB) error {
		const (
			mbiStart, mbiEnd   = 0, 11
			hicnStart, hicnEnd = 11, 22
		)
		cclfBeneficiary := &models.CCLFBeneficiary{
			FileID: fileID,
			MBI:    string(bytes.TrimSpace(b[mbiStart:mbiEnd])),
			HICN:   string(bytes.TrimSpace(b[hicnStart:hicnEnd])),
		}
		err := db.Create(cclfBeneficiary).Error
		if err != nil {
			fmt.Println("Could not create CCLF8 beneficiary record.")
			err = errors.Wrap(err, "could not create CCLF8 beneficiary record")
			log.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func importCCLF9(fileMetadata *cclfFileMetadata) error {
	err := importCCLF(fileMetadata, func(fileID uint, b []byte, db *gorm.DB) error {
		const (
			currIDStart, currIDEnd               = 1, 12
			prevIDStart, prevIDEnd               = 12, 23
			prevIDEffDateStart, prevIDEffDateEnd = 23, 33
			prevIDObsDateStart, prevIDObsDateEnd = 33, 43
		)

		cclf9 := models.CCLFBeneficiaryXref{
			FileID:        fileID,
			XrefIndicator: string(b[0:1]),
			CurrentNum:    string(b[currIDStart:currIDEnd]),
			PrevNum:       string(b[prevIDStart:prevIDEnd]),
			PrevsEfctDt:   string(b[prevIDEffDateStart:prevIDEffDateEnd]),
			PrevsObsltDt:  string(b[prevIDObsDateStart:prevIDObsDateEnd]),
		}
		err := db.Create(&cclf9).Error
		if err != nil {
			fmt.Println("Could not create CCLF9 cross reference record.")
			err = errors.Wrap(err, "could not create CCLF9 cross reference record")
			log.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func importCCLF(fileMetadata *cclfFileMetadata, importFunc func(uint, []byte, *gorm.DB) error) error {
	if fileMetadata == nil {
		fmt.Println("CCLF file not found.")
		err := errors.New("CCLF file not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Importing CCLF%d file %s...\n", fileMetadata.cclfNum, fileMetadata.name)
	log.Infof("Importing CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata.name)

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read CCLF%d archive %s.\n", fileMetadata.cclfNum, fileMetadata.name)
		err := errors.Wrapf(err, "could not read CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.name)
		log.Error(err)
		return err
	}
	defer r.Close()

	if len(r.File) < 1 {
		fmt.Printf("No files found in CCLF%d archive %s.\n", fileMetadata.cclfNum, fileMetadata.name)
		err := fmt.Errorf("no files found in CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.name)
		log.Error(err)
		return err
	}

	cclfFile := models.CCLFFile{
		CCLFNum:         fileMetadata.cclfNum,
		Name:            fileMetadata.name,
		ACOCMSID:        fileMetadata.acoID,
		Timestamp:       fileMetadata.timestamp,
		PerformanceYear: fileMetadata.perfYear,
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err = db.Create(&cclfFile).Error
	if err != nil {
		fmt.Printf("Could not create CCLF%d file record.\n", fileMetadata.cclfNum)
		err = errors.Wrapf(err, "could not create CCLF%d file record", fileMetadata.cclfNum)
		log.Error(err)
		return err
	}

	for i, f := range r.File {
		fmt.Printf("Reading file #%d from archive %s.\n", i, fileMetadata.name)
		log.Infof("Reading file #%d from archive %s", i, fileMetadata.name)
		rc, err := f.Open()
		if err != nil {
			fmt.Printf("Could not read file %s in CCLF%d archive %s.\n", f.Name, fileMetadata.cclfNum, fileMetadata.name)
			err = errors.Wrapf(err, "could not read file %s in CCLF%d archive %s", f.Name, fileMetadata.cclfNum, fileMetadata.name)
			log.Error(err)
			return err
		}
		defer rc.Close()
		sc := bufio.NewScanner(rc)
		for sc.Scan() {
			b := sc.Bytes()
			if len(bytes.TrimSpace(b)) > 0 {
				err = importFunc(cclfFile.ID, b, db)
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
	}

	fmt.Printf("Successfully imported CCLF%d file %s.\n", fileMetadata.cclfNum, fileMetadata.name)
	log.Infof("Successfully imported CCLF%d file %s.", fileMetadata.cclfNum, fileMetadata.name)

	return nil
}

func getCCLFFileMetadata(filePath string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF filename convention for SSP: P.A****.ACO.ZC*Y**.Dyymmdd.Thhmmsst
	// Prefix: T = test, P = prod; A**** = ACO ID; ZC* = CCLF file number; Y** = performance year
	filenameRegexp := regexp.MustCompile(`(T|P)\.(A\d{4})\.ACO\.ZC(0|8|9)Y(\d{2})\.(D\d{6}\.T\d{6})\d`)
	filenameMatches := filenameRegexp.FindStringSubmatch(filePath)
	if len(filenameMatches) < 5 {
		fmt.Printf("Invalid filename for file: %s.\n", filePath)
		err := fmt.Errorf("invalid filename for file: %s", filePath)
		log.Error(err)
		return metadata, err
	}

	cclfNum, err := strconv.Atoi(filenameMatches[3])
	if err != nil {
		fmt.Printf("Failed to parse cclf num from file: %s.\n", filePath)
		err = errors.Wrapf(err, "failed to parse cclf num from file: %s", filePath)
		log.Error(err)
		return metadata, err
	}

	perfYear, err := strconv.Atoi(filenameMatches[4])
	if err != nil {
		fmt.Printf("Failed to parse performance year from file: %s.\n", filePath)
		err = errors.Wrapf(err, "failed to parse performance year from file: %s", filePath)
		log.Error(err)
		return metadata, err
	}

	filenameDate := filenameMatches[5]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		fmt.Printf("Failed to parse date '%s' from file: %s.\n", filenameDate, filePath)
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, filePath)
		log.Error(err)
		return metadata, err
	}

	if filenameMatches[1] == "T" {
		metadata.env = "test"
	} else if filenameMatches[1] == "P" {
		metadata.env = "production"
	}

	metadata.name = filenameMatches[0]
	metadata.cclfNum = cclfNum
	metadata.acoID = filenameMatches[2]
	metadata.timestamp = t
	metadata.perfYear = perfYear

	return metadata, nil
}

func ImportCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	var cclfmap = make(map[string][]*cclfFileMetadata)

	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	if err != nil {
		return 0, 0, 0, err
	}

	if len(cclfmap) == 0 {
		err := errors.New("failed to find any CCLF files in directory")
		log.Error(err)
		return 0, 0, 0, err
	}

	for _, cclflist := range cclfmap {
		var cclf0, cclf8, cclf9 *cclfFileMetadata
		for _, cclf := range cclflist {
			if cclf.cclfNum == 0 {
				cclf0 = cclf
			} else if cclf.cclfNum == 8 {
				cclf8 = cclf
			} else if cclf.cclfNum == 9 {
				cclf9 = cclf
			}
		}
		cclfvalidator, err := importCCLF0(cclf0)
		if err != nil {
			fmt.Printf("Failed to import CCLF0 file: %#v, Skipping CCLF8 file: %#v and CCLF9 file: %#v.\n ", cclf0, cclf8, cclf9)
			log.Errorf("Failed to import CCLF0 file: %#v, Skipping CCLF8 file: %#v and CCLF9 file: %#v ", cclf0, cclf8, cclf9)
			failure++
			skipped += 2
			continue
		} else {
			success++
		}
		err = validate(cclf8, cclfvalidator)
		if err != nil {
			fmt.Printf("Failed to validate CCLF8 file: %#v.\n", cclf8)
			log.Errorf("Failed to validate CCLF8 file: %#v", cclf8)
			failure++
		} else {
			if err = importCCLF8(cclf8); err != nil {
				fmt.Printf("Failed to import CCLF8 file: %#v.\n", cclf8)
				log.Errorf("Failed to import CCLF8 file: %#v ", cclf8)
				failure++
			} else {
				cclf8.imported = true
				success++
			}
		}
		err = validate(cclf9, cclfvalidator)
		if err != nil {
			fmt.Printf("Failed to validate CCLF9 file: %#v.\n", cclf9)
			log.Errorf("Failed to validate CCLF9 file: %#v", cclf9)
			failure++
		} else {
			if err = importCCLF9(cclf9); err != nil {
				fmt.Printf("Failed to import CCLF9 file: %#v.\n", cclf9)
				log.Errorf("Failed to import CCLF9 file: %#v ", cclf9)
				failure++
			} else {
				cclf9.imported = true
				success++
			}
		}
		cclf0.imported = cclf8 != nil && cclf8.imported && cclf9 != nil && cclf9.imported
	}
	cleanupCCLF(cclfmap)

	if failure > 0 {
		err = errors.New("one or more files failed to import correctly")
		log.Error(err)
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func sortCCLFFiles(cclfmap *map[string][]*cclfFileMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error in sorting cclf file: %v.\n", info.Name())
			err = errors.Wrapf(err, "error in sorting cclf file: %v,", info.Name())
			log.Error(err)
			return err
		}
		// Directories are not CCLF files
		if info.IsDir() {
			return nil
		}
		metadata, err := getCCLFFileMetadata(info.Name())
		metadata.filePath = path
		metadata.deliveryDate = info.ModTime()
		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			fmt.Printf("Unknown file found: %#v.\n", metadata)
			log.Errorf("Unknown file found: %#v", metadata)
			*skipped = *skipped + 1

			newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), info.Name())
			err = os.Rename(metadata.filePath, newpath)
			if err != nil {
				fmt.Printf("Error moving unknown file: %#v to pending deletion dir.\n", metadata)
				err = fmt.Errorf("error moving unknown file: %#v to pending deletion dir", metadata)
				log.Error(err)
				return err
			}
			return nil
		}

		// if we get multiple sets of files relating to the same aco for attribution purposes, concat the year
		key := fmt.Sprintf(metadata.acoID+"_%d", metadata.perfYear)
		if (*cclfmap)[key] != nil {
			(*cclfmap)[key] = append((*cclfmap)[key], &metadata)
		} else {
			(*cclfmap)[key] = []*cclfFileMetadata{&metadata}
		}
		return nil
	}
}

func validate(fileMetadata *cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata == nil {
		fmt.Printf("File not found.\n")
		err := errors.New("file not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Validating CCLF%d file %s...\n", fileMetadata.cclfNum, fileMetadata.name)
	log.Infof("Validating CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata.name)

	var key string
	if fileMetadata.cclfNum == 8 {
		key = "CCLF8"
	} else if fileMetadata.cclfNum == 9 {
		key = "CCLF9"
	} else {
		fmt.Printf("Unknown file type when validating file: %#v.\n", fileMetadata)
		err := fmt.Errorf("unknown file type when validating file: %#v", fileMetadata)
		log.Error(err)
		return err
	}

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read archive %s.\n", fileMetadata.name)
		err := errors.Wrapf(err, "could not read archive %s", fileMetadata.name)
		log.Error(err)
		return err
	}
	defer r.Close()

	count := 0
	validator := cclfFileValidator[key]

	for i, f := range r.File {
		fmt.Printf("Reading file #%d from archive %s.\n", i, fileMetadata.name)
		log.Infof("Reading file #%d from archive %s", i, fileMetadata.name)
		rc, err := f.Open()
		if err != nil {
			fmt.Printf("Could not read file %s in archive %s.\n", f.Name, fileMetadata.name)
			err = errors.Wrapf(err, "could not read file %s in archive %s", f.Name, fileMetadata.name)
			log.Error(err)
			return err
		}
		defer rc.Close()
		sc := bufio.NewScanner(rc)
		for sc.Scan() {
			b := sc.Bytes()
			bytelength := len(bytes.TrimSpace(b))
			if bytelength > 0 && bytelength <= validator.maxRecordLength {
				count++

				// currently only errors if there are more records than we expect.
				if count > validator.totalRecordCount {
					fmt.Printf("Maximum record count reached for file %s, Expected record count: %d, Actual record count: %d.\n", key, validator.totalRecordCount, count)
					err := fmt.Errorf("maximum record count reached for file %s, Expected record count: %d, Actual record count: %d ", key, validator.totalRecordCount, count)
					log.Error(err)
					return err
				}
			} else {
				fmt.Printf("Incorrect record length for file %s, Expected record length: %d, Actual record length: %d.\n", key, validator.maxRecordLength, bytelength)
				err := fmt.Errorf("incorrect record length for file %s, Expected record length: %d, Actual record length: %d", key, validator.maxRecordLength, bytelength)
				log.Error(err)
				return err
			}
		}
	}
	fmt.Printf("Successfully validated CCLF%d file %s.\n", fileMetadata.cclfNum, fileMetadata.name)
	log.Infof("Successfully validated CCLF%d file %s.", fileMetadata.cclfNum, fileMetadata.name)
	return nil
}

func DeleteDirectoryContents(dirToDelete string) (filesDeleted int, err error) {
	fmt.Printf("Preparing to delete directory %v.\n", dirToDelete)
	log.Infof("Preparing to delete directory %v", dirToDelete)
	f, err := os.Open(filepath.Clean(dirToDelete))
	if err != nil {
		fmt.Printf("Could not open dir: %s.\n", dirToDelete)
		err = errors.Wrapf(err, "could not open dir: %s", dirToDelete)
		log.Error(err)
		return 0, err
	}
	files, err := f.Readdir(-1)
	if err != nil {
		fmt.Printf("Error reading files from dir: %s.\n", f.Name())
		err = errors.Wrapf(err, "error reading files from dir: %s", f.Name())
		log.Error(err)
		return 0, err
	}
	err = f.Close()
	if err != nil {
		fmt.Printf("Error closing dir: %s.\n", f.Name())
		err = errors.Wrapf(err, "error closing dir: %s", f.Name())
		log.Error(err)
		return 0, err
	}

	for _, file := range files {
		fmt.Printf("Deleting %s.\n", file.Name())
		log.Infof("deleting %s ", file.Name())
		err = os.Remove(filepath.Join(dirToDelete, file.Name()))
		if err != nil {
			fmt.Printf("Error deleting file: %s from dir: %s.\n", file.Name(), dirToDelete)
			err = errors.Wrapf(err, "error deleting file: %s from dir: %s", file.Name(), dirToDelete)
			log.Error(err)
			return 0, err
		}
	}

	fmt.Printf("Successfully deleted all files from dir: %s.\n", dirToDelete)
	log.Infof("Successfully deleted all files from dir: %s", dirToDelete)
	return len(files), nil
}

func cleanupCCLF(cclfmap map[string][]*cclfFileMetadata) {
	for _, cclflist := range cclfmap {
		for _, cclf := range cclflist {
			fmt.Printf("Cleaning up file %#v.\n", cclf)
			log.Infof("Cleaning up file %#v", cclf)
			newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), cclf.name)
			if !cclf.imported {
				// check the timestamp on the failed files
				elapsed := time.Since(cclf.deliveryDate).Hours()
				if int(elapsed) > deleteThresholdHr {
					err := os.Rename(cclf.filePath, newpath)
					if err != nil {
						fmt.Printf("File: %#v failed to cleanup properly.\n", cclf)
						log.Errorf("File: %#v failed to cleanup properly", cclf)
					} else {
						fmt.Printf("File: %#v never ingested, moved to the pending deletion dir.\n", cclf)
						log.Infof("File: %#v never ingested, moved to the pending deletion dir", cclf)
					}
				}
			} else {
				// move the successful files to the deletion dir
				err := os.Rename(cclf.filePath, newpath)
				if err != nil {
					fmt.Printf("File: %#v failed to cleanup properly.\n", cclf)
					log.Errorf("File: %#v failed to cleanup properly", cclf)
				} else {
					fmt.Printf("File: %#v successfully ingested, moved to the pending deletion dir.\n", cclf)
					log.Infof("File: %#v successfully ingested, moved to the pending deletion dir", cclf)
				}
			}
		}
	}
}