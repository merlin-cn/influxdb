import _ from 'lodash'
import React, {PureComponent} from 'react'

import {Form, Columns} from 'src/clockface'
import FluxEditor from 'src/shared/components/FluxEditor'

interface Props {
  script: string
  onChange: (script) => void
}

export default class TaskForm extends PureComponent<Props> {
  public render() {
    return (
      <>
        <Form style={{height: '90%'}}>
          <Form.Element
            label="Script"
            colsXS={Columns.Six}
            offsetXS={Columns.Three}
            errorMessage={''}
          >
            <FluxEditor
              script={this.props.script}
              onChangeScript={this.props.onChange}
              visibility={'visible'}
              status={{text: '', type: ''}}
              suggestions={[]}
            />
          </Form.Element>
        </Form>
      </>
    )
  }
}