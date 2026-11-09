using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000070 RID: 112
	public class FglTest : FglBaseObject
	{
		// Token: 0x17000123 RID: 291
		// (get) Token: 0x0600044C RID: 1100 RVA: 0x00013ACC File Offset: 0x00011CCC
		// (set) Token: 0x0600044D RID: 1101 RVA: 0x00013AD4 File Offset: 0x00011CD4
		private FglTestNode Parent { get; set; }

		// Token: 0x0600044E RID: 1102 RVA: 0x00013AE0 File Offset: 0x00011CE0
		public FglTest(FglTestNode parent)
		{
			this.Parent = parent;
			this.Description = string.Empty;
			this.Input = string.Empty;
			this.Output = string.Empty;
			this.PosNeg = string.Empty;
			this.ErrCode = string.Empty;
			this.ErrDescription = string.Empty;
			this.Memo = string.Empty;
		}

		// Token: 0x17000124 RID: 292
		// (get) Token: 0x0600044F RID: 1103 RVA: 0x00013B52 File Offset: 0x00011D52
		// (set) Token: 0x06000450 RID: 1104 RVA: 0x00013B88 File Offset: 0x00011D88
		public string Order
		{
			get
			{
				return this._order;
			}
			set
			{
				IEnumerable<FglTest> enumerable = this.Parent.FglTests.Where<FglTest>((FglTest p) => p.Order == value && p != this);
				if (enumerable.Count<FglTest>() > 0)
				{
					throw new Exception("順序值不可重複");
				}
				if (string.IsNullOrWhiteSpace(value))
				{
					throw new Exception("順序值不可為空");
				}
				this._order = value;
				this.OnPropertySettingChanged("Order");
			}
		}

		// Token: 0x17000125 RID: 293
		// (get) Token: 0x06000451 RID: 1105 RVA: 0x00013C09 File Offset: 0x00011E09
		// (set) Token: 0x06000452 RID: 1106 RVA: 0x00013C11 File Offset: 0x00011E11
		public string Description
		{
			get
			{
				return this._description;
			}
			set
			{
				this._description = value;
				this.OnPropertySettingChanged("Description");
			}
		}

		// Token: 0x17000126 RID: 294
		// (get) Token: 0x06000453 RID: 1107 RVA: 0x00013C25 File Offset: 0x00011E25
		// (set) Token: 0x06000454 RID: 1108 RVA: 0x00013C2D File Offset: 0x00011E2D
		public string Input
		{
			get
			{
				return this._input;
			}
			set
			{
				this._input = value;
				this.OnPropertySettingChanged("Input");
			}
		}

		// Token: 0x17000127 RID: 295
		// (get) Token: 0x06000455 RID: 1109 RVA: 0x00013C41 File Offset: 0x00011E41
		// (set) Token: 0x06000456 RID: 1110 RVA: 0x00013C49 File Offset: 0x00011E49
		public string Output
		{
			get
			{
				return this._output;
			}
			set
			{
				this._output = value;
				this.OnPropertySettingChanged("Output");
			}
		}

		// Token: 0x17000128 RID: 296
		// (get) Token: 0x06000457 RID: 1111 RVA: 0x00013C5D File Offset: 0x00011E5D
		// (set) Token: 0x06000458 RID: 1112 RVA: 0x00013C65 File Offset: 0x00011E65
		public string PosNeg
		{
			get
			{
				return this._posNeg;
			}
			set
			{
				this._posNeg = value;
				this.OnPropertySettingChanged("PosNeg");
			}
		}

		// Token: 0x17000129 RID: 297
		// (get) Token: 0x06000459 RID: 1113 RVA: 0x00013C79 File Offset: 0x00011E79
		// (set) Token: 0x0600045A RID: 1114 RVA: 0x00013C81 File Offset: 0x00011E81
		public string ErrCode
		{
			get
			{
				return this._errCode;
			}
			set
			{
				this._errCode = value;
				this.OnPropertySettingChanged("ErrCode");
			}
		}

		// Token: 0x1700012A RID: 298
		// (get) Token: 0x0600045B RID: 1115 RVA: 0x00013C95 File Offset: 0x00011E95
		// (set) Token: 0x0600045C RID: 1116 RVA: 0x00013C9D File Offset: 0x00011E9D
		public string ErrDescription
		{
			get
			{
				return this._errDescription;
			}
			set
			{
				this._errDescription = value;
				this.OnPropertySettingChanged("ErrDescription");
			}
		}

		// Token: 0x1700012B RID: 299
		// (get) Token: 0x0600045D RID: 1117 RVA: 0x00013CB1 File Offset: 0x00011EB1
		// (set) Token: 0x0600045E RID: 1118 RVA: 0x00013CB9 File Offset: 0x00011EB9
		public string Memo
		{
			get
			{
				return this._memo;
			}
			set
			{
				this._memo = value;
				this.OnPropertySettingChanged("Memo");
			}
		}

		// Token: 0x0600045F RID: 1119 RVA: 0x00013CD0 File Offset: 0x00011ED0
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Parent.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Parent.Status = CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x06000460 RID: 1120 RVA: 0x00013D28 File Offset: 0x00011F28
		public static FglTest Parse(FglTestNode parent, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			FglTest fglTest = new FglTest(parent);
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string localName;
				switch (localName = xattribute.Name.LocalName)
				{
				case "order":
					fglTest.Order = xattribute.Value;
					continue;
				case "desc":
					fglTest.Description = xattribute.Value;
					continue;
				case "input":
					fglTest.Input = xattribute.Value;
					continue;
				case "output":
					fglTest.Output = xattribute.Value;
					continue;
				case "pos_neg":
					fglTest.PosNeg = xattribute.Value;
					continue;
				case "err_code":
					fglTest.ErrCode = xattribute.Value;
					continue;
				case "err_desc":
					fglTest.ErrDescription = xattribute.Value;
					continue;
				case "memo":
					fglTest.Memo = xattribute.Value;
					continue;
				}
				fglTest._attributes.Add(xattribute.Name.LocalName, xattribute.Value);
			}
			return fglTest;
		}

		// Token: 0x06000461 RID: 1121 RVA: 0x00013EE0 File Offset: 0x000120E0
		public XElement ToXElement()
		{
			XElement xelement = new XElement("scenario");
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			xelement.SetAttributeValue("order", this.Order);
			xelement.SetAttributeValue("desc", this.Description);
			xelement.SetAttributeValue("input", this.Input);
			xelement.SetAttributeValue("output", this.Output);
			xelement.SetAttributeValue("pos_neg", this.PosNeg);
			xelement.SetAttributeValue("err_code", this.ErrCode);
			xelement.SetAttributeValue("err_desc", this.ErrDescription);
			xelement.SetAttributeValue("memo", this.Memo);
			return xelement;
		}

		// Token: 0x040001AC RID: 428
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040001AD RID: 429
		private string _order;

		// Token: 0x040001AE RID: 430
		private string _description;

		// Token: 0x040001AF RID: 431
		private string _input;

		// Token: 0x040001B0 RID: 432
		private string _output;

		// Token: 0x040001B1 RID: 433
		private string _posNeg;

		// Token: 0x040001B2 RID: 434
		private string _errCode;

		// Token: 0x040001B3 RID: 435
		private string _errDescription;

		// Token: 0x040001B4 RID: 436
		private string _memo;
	}
}
