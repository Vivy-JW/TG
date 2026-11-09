using System;
using System.ComponentModel;

namespace SpecDesigner.FormEditor.Relation
{
	// Token: 0x02000044 RID: 68
	public class TableComboBoxItem : INotifyPropertyChanged
	{
		// Token: 0x14000007 RID: 7
		// (add) Token: 0x0600028B RID: 651 RVA: 0x0000DA40 File Offset: 0x0000BC40
		// (remove) Token: 0x0600028C RID: 652 RVA: 0x0000DA78 File Offset: 0x0000BC78
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x17000081 RID: 129
		// (get) Token: 0x0600028D RID: 653 RVA: 0x0000DAAD File Offset: 0x0000BCAD
		// (set) Token: 0x0600028E RID: 654 RVA: 0x0000DAB5 File Offset: 0x0000BCB5
		public bool IsEnabled
		{
			get
			{
				return this._isEnabled;
			}
			set
			{
				this._isEnabled = value;
				this.NotifyPropertyChanged("IsEnabled");
			}
		}

		// Token: 0x17000082 RID: 130
		// (get) Token: 0x0600028F RID: 655 RVA: 0x0000DAC9 File Offset: 0x0000BCC9
		// (set) Token: 0x06000290 RID: 656 RVA: 0x0000DAD1 File Offset: 0x0000BCD1
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				this._name = value;
				this.NotifyPropertyChanged("Name");
			}
		}

		// Token: 0x17000083 RID: 131
		// (get) Token: 0x06000291 RID: 657 RVA: 0x0000DAE5 File Offset: 0x0000BCE5
		// (set) Token: 0x06000292 RID: 658 RVA: 0x0000DAED File Offset: 0x0000BCED
		public string Text
		{
			get
			{
				return this._text;
			}
			set
			{
				this._text = value;
				this.NotifyPropertyChanged("Text");
			}
		}

		// Token: 0x06000293 RID: 659 RVA: 0x0000DB01 File Offset: 0x0000BD01
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x04000173 RID: 371
		private bool _isEnabled;

		// Token: 0x04000174 RID: 372
		private string _name;

		// Token: 0x04000175 RID: 373
		private string _text;
	}
}
