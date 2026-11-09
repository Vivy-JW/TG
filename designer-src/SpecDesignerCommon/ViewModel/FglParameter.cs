using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000096 RID: 150
	public class FglParameter : FglBaseObject
	{
		// Token: 0x170001BD RID: 445
		// (get) Token: 0x06000613 RID: 1555 RVA: 0x0001B9B1 File Offset: 0x00019BB1
		// (set) Token: 0x06000614 RID: 1556 RVA: 0x0001B9B9 File Offset: 0x00019BB9
		public FglParameterNode Parent { get; set; }

		// Token: 0x170001BE RID: 446
		// (get) Token: 0x06000615 RID: 1557 RVA: 0x0001B9C2 File Offset: 0x00019BC2
		// (set) Token: 0x06000616 RID: 1558 RVA: 0x0001B9CA File Offset: 0x00019BCA
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				this._name = value;
				this.OnPropertySettingChanged("Name");
			}
		}

		// Token: 0x170001BF RID: 447
		// (get) Token: 0x06000617 RID: 1559 RVA: 0x0001B9DE File Offset: 0x00019BDE
		// (set) Token: 0x06000618 RID: 1560 RVA: 0x0001B9E6 File Offset: 0x00019BE6
		public bool IsReturn { get; set; }

		// Token: 0x170001C0 RID: 448
		// (get) Token: 0x06000619 RID: 1561 RVA: 0x0001B9EF File Offset: 0x00019BEF
		// (set) Token: 0x0600061A RID: 1562 RVA: 0x0001BA48 File Offset: 0x00019C48
		public string Order
		{
			get
			{
				return this._order;
			}
			set
			{
				IEnumerable<FglParameter> enumerable = (this.IsReturn ? this.Parent.Returns.Where<FglParameter>((FglParameter p) => p.Order == value && p != this) : this.Parent.Inputs.Where<FglParameter>((FglParameter p) => p.Order == value && p != this));
				if (enumerable.Count<FglParameter>() > 0)
				{
					throw new Exception("Order不可重複");
				}
				this._order = value;
				this.OnPropertySettingChanged("Order");
			}
		}

		// Token: 0x170001C1 RID: 449
		// (get) Token: 0x0600061B RID: 1563 RVA: 0x0001BAD8 File Offset: 0x00019CD8
		public string Type
		{
			get
			{
				if (string.IsNullOrWhiteSpace(this.DB) && string.IsNullOrWhiteSpace(this.Col))
				{
					return string.Empty;
				}
				if (!string.IsNullOrEmpty(this.DB) && string.IsNullOrEmpty(this.Col))
				{
					return this.DB;
				}
				return string.Format("{0}.{1}", this.DB, this.Col);
			}
		}

		// Token: 0x170001C2 RID: 450
		// (get) Token: 0x0600061C RID: 1564 RVA: 0x0001BB3C File Offset: 0x00019D3C
		public string ReferenceType
		{
			get
			{
				if (string.IsNullOrWhiteSpace(this.DB) && string.IsNullOrWhiteSpace(this.Col))
				{
					return string.Empty;
				}
				if (!string.IsNullOrEmpty(this.DB) && string.IsNullOrEmpty(this.Col))
				{
					return this.DB;
				}
				return string.Format("LIKE {0}.{1}", this.DB, this.Col);
			}
		}

		// Token: 0x170001C3 RID: 451
		// (get) Token: 0x0600061D RID: 1565 RVA: 0x0001BBA0 File Offset: 0x00019DA0
		// (set) Token: 0x0600061E RID: 1566 RVA: 0x0001BBC0 File Offset: 0x00019DC0
		public string DB
		{
			get
			{
				return this._db;
			}
			set
			{
				if (string.IsNullOrWhiteSpace(value))
				{
					throw new Exception("DB不可為空");
				}
				this._db = value;
				this.OnPropertySettingChanged("DB");
				XElement xelement = TableColumnHelper.FindTableColumns(this._db);
				if (xelement == null)
				{
					this.Col = string.Empty;
					return;
				}
				string text = (from ele in xelement.Elements("column")
					select ele.Attribute("name").Value).ElementAtOrDefault<string>(0);
				this.Col = ((text == null) ? string.Empty : text);
			}
		}

		// Token: 0x170001C4 RID: 452
		// (get) Token: 0x0600061F RID: 1567 RVA: 0x0001BC57 File Offset: 0x00019E57
		// (set) Token: 0x06000620 RID: 1568 RVA: 0x0001BC60 File Offset: 0x00019E60
		public string Col
		{
			get
			{
				return this._col;
			}
			set
			{
				if (TableColumnHelper.FindTableColumns(this.DB) != null && string.IsNullOrWhiteSpace(value))
				{
					throw new Exception("Column不可為空");
				}
				this._col = value;
				this.OnPropertySettingChanged("Col");
				if (this.Name == this.DefaultName)
				{
					this.Name = string.Format("{0}_{1}", this.IsReturn ? "r" : "p", this.Col);
				}
			}
		}

		// Token: 0x170001C5 RID: 453
		// (get) Token: 0x06000621 RID: 1569 RVA: 0x0001BCDC File Offset: 0x00019EDC
		// (set) Token: 0x06000622 RID: 1570 RVA: 0x0001BCE4 File Offset: 0x00019EE4
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

		// Token: 0x170001C6 RID: 454
		// (get) Token: 0x06000623 RID: 1571 RVA: 0x0001BCF8 File Offset: 0x00019EF8
		// (set) Token: 0x06000624 RID: 1572 RVA: 0x0001BD00 File Offset: 0x00019F00
		public string Purpose
		{
			get
			{
				return this._purpose;
			}
			set
			{
				this._purpose = value;
				this.OnPropertySettingChanged("Purpose");
			}
		}

		// Token: 0x06000625 RID: 1573 RVA: 0x0001BD14 File Offset: 0x00019F14
		public FglParameter(FglParameterNode parent, string name, bool isReturn)
		{
			this.Parent = parent;
			this.Name = name;
			this.IsReturn = isReturn;
			this.Description = string.Empty;
			this.Purpose = string.Empty;
		}

		// Token: 0x06000626 RID: 1574 RVA: 0x0001BD47 File Offset: 0x00019F47
		public FglParameter(string name, bool isReturn)
			: this(null, name, isReturn)
		{
		}

		// Token: 0x06000627 RID: 1575 RVA: 0x0001BD52 File Offset: 0x00019F52
		public FglParameter(FglParameterNode parent, bool isReturn)
			: this(parent, string.Empty, isReturn)
		{
			this.Name = this.DefaultName;
		}

		// Token: 0x170001C7 RID: 455
		// (get) Token: 0x06000628 RID: 1576 RVA: 0x0001BD6D File Offset: 0x00019F6D
		private string DefaultName
		{
			get
			{
				return string.Format("{0}_{1}", this.IsReturn ? "r" : "p", "parameter");
			}
		}

		// Token: 0x170001C8 RID: 456
		// (get) Token: 0x06000629 RID: 1577 RVA: 0x0001BD92 File Offset: 0x00019F92
		public string NameWithPurpose
		{
			get
			{
				return string.Format("{0}  {1}", this.Name, this.Purpose);
			}
		}

		// Token: 0x0600062A RID: 1578 RVA: 0x0001BDAC File Offset: 0x00019FAC
		public XElement ToXElement()
		{
			XElement xelement;
			if (this.IsReturn)
			{
				xelement = new XElement("output");
			}
			else
			{
				xelement = new XElement("input");
			}
			xelement.SetAttributeValue("order", this.Order);
			xelement.SetAttributeValue("name", this.Name);
			xelement.SetAttributeValue("type", this.Type);
			xelement.SetAttributeValue("desc", this.Description);
			xelement.SetAttributeValue("purpose", this.Purpose);
			return xelement;
		}

		// Token: 0x0600062B RID: 1579 RVA: 0x0001BE54 File Offset: 0x0001A054
		public static FglParameter Parse(FglParameterNode parent, XElement paramElement)
		{
			if (paramElement == null)
			{
				return null;
			}
			FglParameter fglParameter = null;
			if (paramElement.Name.LocalName.ToUpper() == "INPUT")
			{
				fglParameter = new FglParameter(parent, false);
			}
			else
			{
				fglParameter = new FglParameter(parent, true);
			}
			foreach (XAttribute xattribute in paramElement.Attributes())
			{
				string text;
				if ((text = xattribute.Name.LocalName.ToLower()) != null)
				{
					if (!(text == "order"))
					{
						if (!(text == "name"))
						{
							if (!(text == "type"))
							{
								if (!(text == "desc"))
								{
									if (text == "purpose")
									{
										fglParameter.Purpose = xattribute.Value;
									}
								}
								else
								{
									fglParameter.Description = xattribute.Value;
								}
							}
							else
							{
								string[] array = xattribute.Value.Split(new char[] { '.' });
								if (array.Count<string>() == 2)
								{
									fglParameter._db = array[0];
									fglParameter._col = array[1];
								}
							}
						}
						else
						{
							fglParameter.Name = xattribute.Value;
						}
					}
					else
					{
						fglParameter.Order = xattribute.Value;
					}
				}
			}
			return fglParameter;
		}

		// Token: 0x0600062C RID: 1580 RVA: 0x0001BFA8 File Offset: 0x0001A1A8
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Parent.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Parent.Status = CodeSpecStatus.MODIFY;
				EventAggregatorManager.Get(this.Parent.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.Parent.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x04000253 RID: 595
		private string _name;

		// Token: 0x04000254 RID: 596
		private string _order;

		// Token: 0x04000255 RID: 597
		private string _db;

		// Token: 0x04000256 RID: 598
		private string _col;

		// Token: 0x04000257 RID: 599
		private string _description;

		// Token: 0x04000258 RID: 600
		private string _purpose;
	}
}
