using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using SpecDesignerCommon.Events;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000048 RID: 72
	public class FglComponent : FglBaseObject
	{
		// Token: 0x1700008E RID: 142
		// (get) Token: 0x0600024F RID: 591 RVA: 0x0000A270 File Offset: 0x00008470
		// (set) Token: 0x06000250 RID: 592 RVA: 0x0000A278 File Offset: 0x00008478
		public FglSpecification Parent { get; private set; }

		// Token: 0x1700008F RID: 143
		// (get) Token: 0x06000251 RID: 593 RVA: 0x0000A281 File Offset: 0x00008481
		public PackageKey ProgramKey
		{
			get
			{
				return this.Parent.ProgramKey;
			}
		}

		// Token: 0x17000090 RID: 144
		// (get) Token: 0x06000252 RID: 594 RVA: 0x0000A28E File Offset: 0x0000848E
		// (set) Token: 0x06000253 RID: 595 RVA: 0x0000A296 File Offset: 0x00008496
		public CodeSpecStatus Status { get; set; }

		// Token: 0x17000091 RID: 145
		// (get) Token: 0x06000254 RID: 596 RVA: 0x0000A29F File Offset: 0x0000849F
		public ReadOnlyObservableCollection<FglParameter> Inputs
		{
			get
			{
				return new ReadOnlyObservableCollection<FglParameter>(this._parameterNode.Inputs);
			}
		}

		// Token: 0x17000092 RID: 146
		// (get) Token: 0x06000255 RID: 597 RVA: 0x0000A2B1 File Offset: 0x000084B1
		public ReadOnlyObservableCollection<FglParameter> Returns
		{
			get
			{
				return new ReadOnlyObservableCollection<FglParameter>(this._parameterNode.Returns);
			}
		}

		// Token: 0x17000093 RID: 147
		// (get) Token: 0x06000256 RID: 598 RVA: 0x0000A2C3 File Offset: 0x000084C3
		public ReadOnlyObservableCollection<FglKeyword> FglKeywords
		{
			get
			{
				if (this._keywordNode == null)
				{
					return null;
				}
				return new ReadOnlyObservableCollection<FglKeyword>(this._keywordNode.FglKeywords);
			}
		}

		// Token: 0x17000094 RID: 148
		// (get) Token: 0x06000257 RID: 599 RVA: 0x0000A2DF File Offset: 0x000084DF
		public ReadOnlyObservableCollection<FglDimension> FglDimensions
		{
			get
			{
				if (this._dimensionNode == null)
				{
					return null;
				}
				return new ReadOnlyObservableCollection<FglDimension>(this._dimensionNode.FglDimensions);
			}
		}

		// Token: 0x17000095 RID: 149
		// (get) Token: 0x06000258 RID: 600 RVA: 0x0000A2FB File Offset: 0x000084FB
		public ReadOnlyObservableCollection<FglFrontComp> FglFrontComps
		{
			get
			{
				if (this._frontCompNode == null)
				{
					return null;
				}
				return new ReadOnlyObservableCollection<FglFrontComp>(this._frontCompNode.FglFrontComps);
			}
		}

		// Token: 0x17000096 RID: 150
		// (get) Token: 0x06000259 RID: 601 RVA: 0x0000A317 File Offset: 0x00008517
		public ReadOnlyObservableCollection<FglTest> FglTests
		{
			get
			{
				if (this._testNode == null)
				{
					return null;
				}
				return new ReadOnlyObservableCollection<FglTest>(this._testNode.FglTests);
			}
		}

		// Token: 0x17000097 RID: 151
		// (get) Token: 0x0600025A RID: 602 RVA: 0x0000A333 File Offset: 0x00008533
		// (set) Token: 0x0600025B RID: 603 RVA: 0x0000A33B File Offset: 0x0000853B
		public FglSpecContent SpecContent { get; private set; }

		// Token: 0x17000098 RID: 152
		// (get) Token: 0x0600025C RID: 604 RVA: 0x0000A344 File Offset: 0x00008544
		// (set) Token: 0x0600025D RID: 605 RVA: 0x0000A3A4 File Offset: 0x000085A4
		public string ID
		{
			get
			{
				return this._id;
			}
			set
			{
				if ((this.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
				{
					if (string.IsNullOrWhiteSpace(value))
					{
						throw new Exception("ID不可為空");
					}
					IEnumerable<FglComponent> enumerable = this.Parent.Components.Where<FglComponent>((FglComponent c) => c.ID == value && c != this && (c.Status & CodeSpecStatus.DELETE) != CodeSpecStatus.DELETE);
					if (enumerable.Count<FglComponent>() > 0)
					{
						throw new Exception("ID不可重複");
					}
					IEnumerable<FglComponent> enumerable2 = SettingManager.Get().GetTzpManger(this.ProgramKey).FglSpecificationInfo.Components.Where<FglComponent>((FglComponent comp) => comp.ID == value && (comp.Status & CodeSpecStatus.DELETE) == CodeSpecStatus.DELETE);
					foreach (FglComponent fglComponent in enumerable2)
					{
						this.Parent.Remove(fglComponent);
					}
					FglComponent fglComponent2 = this.Clone();
					fglComponent2.Status = CodeSpecStatus.DELETE;
					this.Parent.Add(fglComponent2);
					if (this._dimensionNode.FglDimensions.Count > 0)
					{
						this._dimensionNode.Status = CodeSpecStatus.MODIFY;
					}
					if (this._parameterNode.Inputs.Count > 0 || this._parameterNode.Returns.Count > 0)
					{
						this._parameterNode.Status = CodeSpecStatus.MODIFY;
					}
					if (this._keywordNode.FglKeywords.Count > 0)
					{
						this._keywordNode.Status = CodeSpecStatus.MODIFY;
					}
					if (this._frontCompNode.FglFrontComps.Count > 0)
					{
						this._frontCompNode.Status = CodeSpecStatus.MODIFY;
					}
					if (this._testNode.FglTests.Count > 0)
					{
						this._testNode.Status = CodeSpecStatus.MODIFY;
					}
					this.SpecContent.Status = CodeSpecStatus.MODIFY;
				}
				this._id = value;
				this.OnPropertySettingChanged("ID");
			}
		}

		// Token: 0x17000099 RID: 153
		// (get) Token: 0x0600025E RID: 606 RVA: 0x0000A598 File Offset: 0x00008798
		// (set) Token: 0x0600025F RID: 607 RVA: 0x0000A5A0 File Offset: 0x000087A0
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

		// Token: 0x1700009A RID: 154
		// (get) Token: 0x06000260 RID: 608 RVA: 0x0000A5B4 File Offset: 0x000087B4
		// (set) Token: 0x06000261 RID: 609 RVA: 0x0000A5BC File Offset: 0x000087BC
		public string ForDB
		{
			get
			{
				return this._forDB;
			}
			set
			{
				this._forDB = value;
				this.OnPropertySettingChanged("ForDB");
			}
		}

		// Token: 0x1700009B RID: 155
		// (get) Token: 0x06000262 RID: 610 RVA: 0x0000A5D0 File Offset: 0x000087D0
		// (set) Token: 0x06000263 RID: 611 RVA: 0x0000A5D8 File Offset: 0x000087D8
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

		// Token: 0x06000264 RID: 612 RVA: 0x0000A5EC File Offset: 0x000087EC
		public FglComponent(FglSpecification parent)
		{
			this.Parent = parent;
			this.Description = string.Empty;
			this.Purpose = string.Empty;
			this.SpecContent = new FglSpecContent(this)
			{
				Status = CodeSpecStatus.LOADED
			};
			this._keywordNode = new FglKeywordNode(this);
			this._parameterNode = new FglParameterNode(this);
			this._frontCompNode = new FglFrontCompNode(this);
			this._dimensionNode = new FglDimensionNode(this);
			this._testNode = new FglTestNode(this);
			this.Status = CodeSpecStatus.NULL;
		}

		// Token: 0x06000265 RID: 613 RVA: 0x0000A6D8 File Offset: 0x000088D8
		public static FglComponent Initial(FglSpecification parent)
		{
			FglComponent fglComponent = new FglComponent(parent);
			string defaultID = string.Format("{0}_", parent.ProgramKey.Program);
			for (;;)
			{
				IEnumerable<FglComponent> enumerable = parent.Components.Where<FglComponent>((FglComponent com) => com.ID == defaultID && (com.Status & CodeSpecStatus.DELETE) == CodeSpecStatus.NULL);
				if (enumerable.Count<FglComponent>() == 0)
				{
					break;
				}
				defaultID = string.Format("{0}1", defaultID);
			}
			fglComponent.ID = defaultID;
			fglComponent.ForDB = ((SettingManager.Get().ForDBs.Count > 0) ? SettingManager.Get().ForDBs[0].Value.ToString() : "0");
			fglComponent.Status = CodeSpecStatus.LOADED | CodeSpecStatus.CREATE;
			return fglComponent;
		}

		// Token: 0x06000266 RID: 614 RVA: 0x0000A7A0 File Offset: 0x000089A0
		public FglComponent Clone()
		{
			XElement xelement = this.ToXElement();
			return FglComponent.Parse(this.Parent, xelement);
		}

		// Token: 0x06000267 RID: 615 RVA: 0x0000A7E4 File Offset: 0x000089E4
		public FglParameter InitialParameter(bool isReturn)
		{
			if (this._parameterNode == null)
			{
				this._parameterNode = new FglParameterNode(this);
			}
			FglParameter fglParameter = new FglParameter(this._parameterNode, isReturn);
			string defaultName = fglParameter.Name;
			ObservableCollection<FglParameter> observableCollection = (isReturn ? this._parameterNode.Returns : this._parameterNode.Inputs);
			for (;;)
			{
				IEnumerable<FglParameter> enumerable = observableCollection.Where<FglParameter>((FglParameter com) => com.Name == defaultName);
				if (enumerable.Count<FglParameter>() == 0)
				{
					break;
				}
				defaultName = string.Format("{0}1", defaultName);
			}
			fglParameter.Name = defaultName;
			ObservableCollection<FglParameter> observableCollection2 = (isReturn ? this._parameterNode.Returns : this._parameterNode.Inputs);
			IOrderedEnumerable<FglParameter> orderedEnumerable = observableCollection2.OrderBy<FglParameter, string>((FglParameter item) => item.Order);
			if (orderedEnumerable.Count<FglParameter>() > 0)
			{
				int num = 1;
				if (int.TryParse(orderedEnumerable.LastOrDefault<FglParameter>().Order, out num))
				{
					fglParameter.Order = (num + 1).ToString();
				}
				else
				{
					fglParameter.Order = orderedEnumerable.LastOrDefault<FglParameter>().Order;
				}
			}
			else
			{
				fglParameter.Order = "1";
			}
			return fglParameter;
		}

		// Token: 0x06000268 RID: 616 RVA: 0x0000A930 File Offset: 0x00008B30
		public FglKeyword InitialKeyword()
		{
			if (this._keywordNode == null)
			{
				this._keywordNode = new FglKeywordNode(this);
			}
			FglKeyword fglKeyword = new FglKeyword(this._keywordNode);
			IOrderedEnumerable<FglKeyword> orderedEnumerable = this._keywordNode.FglKeywords.OrderBy<FglKeyword, string>((FglKeyword item) => item.Order);
			if (orderedEnumerable.Count<FglKeyword>() > 0)
			{
				int num = 1;
				if (int.TryParse(orderedEnumerable.LastOrDefault<FglKeyword>().Order, out num))
				{
					fglKeyword.Order = (num + 1).ToString();
				}
				else
				{
					fglKeyword.Order = orderedEnumerable.LastOrDefault<FglKeyword>().Order;
				}
			}
			else
			{
				fglKeyword.Order = "1";
			}
			fglKeyword.String = fglKeyword.Order.ToString();
			return fglKeyword;
		}

		// Token: 0x06000269 RID: 617 RVA: 0x0000A9F8 File Offset: 0x00008BF8
		public FglFrontComp InitialFrontComp()
		{
			if (this._frontCompNode == null)
			{
				this._frontCompNode = new FglFrontCompNode(this);
			}
			FglFrontComp fglFrontComp = new FglFrontComp(this._frontCompNode);
			IOrderedEnumerable<FglFrontComp> orderedEnumerable = this._frontCompNode.FglFrontComps.OrderBy<FglFrontComp, string>((FglFrontComp item) => item.Order);
			if (orderedEnumerable.Count<FglFrontComp>() > 0)
			{
				int num = 1;
				if (int.TryParse(orderedEnumerable.LastOrDefault<FglFrontComp>().Order, out num))
				{
					fglFrontComp.Order = (num + 1).ToString();
				}
				else
				{
					fglFrontComp.Order = orderedEnumerable.LastOrDefault<FglFrontComp>().Order;
				}
			}
			else
			{
				fglFrontComp.Order = "1";
			}
			return fglFrontComp;
		}

		// Token: 0x0600026A RID: 618 RVA: 0x0000AAAC File Offset: 0x00008CAC
		public FglDimension InitialDimension()
		{
			if (this._dimensionNode == null)
			{
				this._dimensionNode = new FglDimensionNode(this);
			}
			FglDimension fglDimension = new FglDimension(this._dimensionNode);
			IOrderedEnumerable<FglDimension> orderedEnumerable = this._dimensionNode.FglDimensions.OrderBy<FglDimension, string>((FglDimension item) => item.No);
			if (orderedEnumerable.Count<FglDimension>() > 0)
			{
				int num = 1;
				if (int.TryParse(orderedEnumerable.LastOrDefault<FglDimension>().No, out num))
				{
					fglDimension.No = (num + 1).ToString();
				}
				else
				{
					fglDimension.No = orderedEnumerable.LastOrDefault<FglDimension>().No;
				}
			}
			else
			{
				fglDimension.No = "1";
			}
			return fglDimension;
		}

		// Token: 0x0600026B RID: 619 RVA: 0x0000AB60 File Offset: 0x00008D60
		public FglTest InitialTest()
		{
			if (this._testNode == null)
			{
				this._testNode = new FglTestNode(this);
			}
			FglTest fglTest = new FglTest(this._testNode);
			IOrderedEnumerable<FglTest> orderedEnumerable = this._testNode.FglTests.OrderBy<FglTest, string>((FglTest item) => item.Order);
			if (orderedEnumerable.Count<FglTest>() > 0)
			{
				int num = 1;
				if (int.TryParse(orderedEnumerable.LastOrDefault<FglTest>().Order, out num))
				{
					fglTest.Order = (num + 1).ToString();
				}
				else
				{
					fglTest.Order = orderedEnumerable.LastOrDefault<FglTest>().Order;
				}
			}
			else
			{
				fglTest.Order = "1";
			}
			return fglTest;
		}

		// Token: 0x0600026C RID: 620 RVA: 0x0000AC0C File Offset: 0x00008E0C
		public void Add(FglParameter parameter)
		{
			if (this._parameterNode == null)
			{
				this._parameterNode = new FglParameterNode(this);
			}
			parameter.Parent = this._parameterNode;
			this._parameterNode.Add(parameter);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x0600026D RID: 621 RVA: 0x0000AC60 File Offset: 0x00008E60
		public void Remove(FglParameter parameter)
		{
			if (this._parameterNode == null)
			{
				return;
			}
			this._parameterNode.Remove(parameter);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x0600026E RID: 622 RVA: 0x0000AC92 File Offset: 0x00008E92
		public void Add(FglDimension dimension)
		{
			if (this._dimensionNode == null)
			{
				this._dimensionNode = new FglDimensionNode(this);
			}
			this._dimensionNode.Add(dimension);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x0600026F RID: 623 RVA: 0x0000ACCF File Offset: 0x00008ECF
		public void Remove(FglDimension dimension)
		{
			if (this._dimensionNode == null)
			{
				return;
			}
			this._dimensionNode.Remove(dimension);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000270 RID: 624 RVA: 0x0000AD01 File Offset: 0x00008F01
		public void Add(FglTest test)
		{
			if (this._testNode == null)
			{
				this._testNode = new FglTestNode(this);
			}
			this._testNode.Add(test);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000271 RID: 625 RVA: 0x0000AD3E File Offset: 0x00008F3E
		public void Remove(FglTest test)
		{
			this._testNode.Remove(test);
			this._testNode.Status = CodeSpecStatus.MODIFY;
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000272 RID: 626 RVA: 0x0000AD73 File Offset: 0x00008F73
		public void Add(FglKeyword keyword)
		{
			if (this._keywordNode == null)
			{
				this._keywordNode = new FglKeywordNode(this);
			}
			this._keywordNode.Add(keyword);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000273 RID: 627 RVA: 0x0000ADB0 File Offset: 0x00008FB0
		public void Remove(FglKeyword keyword)
		{
			if (this._keywordNode == null)
			{
				return;
			}
			this._keywordNode.Remove(keyword);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000274 RID: 628 RVA: 0x0000ADE4 File Offset: 0x00008FE4
		public void Add(FglFrontComp frontComp)
		{
			if (this._frontCompNode == null)
			{
				this._frontCompNode = new FglFrontCompNode(this);
			}
			frontComp.Parent = this._frontCompNode;
			this._frontCompNode.Add(frontComp);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000275 RID: 629 RVA: 0x0000AE38 File Offset: 0x00009038
		public void Remove(FglFrontComp frontComp)
		{
			if (this._frontCompNode == null)
			{
				return;
			}
			this._frontCompNode.Remove(frontComp);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x06000276 RID: 630 RVA: 0x0000AE6C File Offset: 0x0000906C
		public static FglComponent Parse(FglSpecification spec, XElement xelement)
		{
			if (xelement == null)
			{
				return null;
			}
			FglComponent fglComponent = new FglComponent(spec);
			CodeSpecStatus codeSpecStatus = CodeSpecStatus.NULL;
			foreach (XAttribute xattribute in xelement.Attributes())
			{
				string text;
				if ((text = xattribute.Name.LocalName.ToLower()) != null)
				{
					if (text == "id")
					{
						fglComponent.ID = xattribute.Value;
						continue;
					}
					if (text == "desc")
					{
						fglComponent.Description = xattribute.Value;
						continue;
					}
					if (text == "for_db")
					{
						fglComponent.ForDB = xattribute.Value;
						continue;
					}
					if (text == "purpose")
					{
						fglComponent.Purpose = xattribute.Value;
						continue;
					}
					if (text == "status")
					{
						if (xattribute.Value.Equals("d", StringComparison.InvariantCultureIgnoreCase))
						{
							codeSpecStatus = CodeSpecStatus.DELETE;
							continue;
						}
						if (xattribute.Value.Equals("u", StringComparison.InvariantCultureIgnoreCase))
						{
							codeSpecStatus = CodeSpecStatus.MODIFY;
							continue;
						}
						if (string.IsNullOrWhiteSpace(xattribute.Value))
						{
							codeSpecStatus = CodeSpecStatus.NULL;
							continue;
						}
						continue;
					}
				}
				fglComponent._attributes.Add(xattribute.Name.LocalName, xattribute.Value);
			}
			foreach (XElement xelement2 in xelement.Elements())
			{
				string text2;
				switch (text2 = xelement2.Name.LocalName.ToLower())
				{
				case "param_rtn":
				{
					FglParameterNode fglParameterNode = FglParameterNode.Parse(fglComponent, xelement2);
					if (fglParameterNode != null)
					{
						fglComponent._parameterNode = fglParameterNode;
					}
					break;
				}
				case "front_comp":
				{
					FglFrontCompNode fglFrontCompNode = FglFrontCompNode.Parse(fglComponent, xelement2);
					if (fglFrontCompNode != null)
					{
						fglComponent._frontCompNode = fglFrontCompNode;
					}
					break;
				}
				case "key_word":
				{
					FglKeywordNode fglKeywordNode = FglKeywordNode.Parse(fglComponent, xelement2);
					if (fglKeywordNode != null)
					{
						fglComponent._keywordNode = fglKeywordNode;
					}
					break;
				}
				case "dimension":
				{
					FglDimensionNode fglDimensionNode = FglDimensionNode.Parse(fglComponent, xelement2);
					if (fglDimensionNode != null)
					{
						fglComponent._dimensionNode = fglDimensionNode;
					}
					break;
				}
				case "test":
				{
					FglTestNode fglTestNode = FglTestNode.Parse(fglComponent, xelement2);
					if (fglTestNode != null)
					{
						fglComponent._testNode = fglTestNode;
					}
					break;
				}
				case "spec":
				{
					FglSpecContent fglSpecContent = FglSpecContent.Parse(fglComponent, xelement2);
					if (fglSpecContent != null)
					{
						fglComponent.SpecContent = fglSpecContent;
					}
					break;
				}
				}
			}
			fglComponent.Status = CodeSpecStatus.LOADED | codeSpecStatus;
			return fglComponent;
		}

		// Token: 0x1700009C RID: 156
		// (get) Token: 0x06000277 RID: 631 RVA: 0x0000B15C File Offset: 0x0000935C
		public string FunctionName
		{
			get
			{
				StringBuilder stringBuilder = new StringBuilder(this.ID);
				stringBuilder.Append("(");
				for (int i = 0; i < this._parameterNode.Inputs.Count; i++)
				{
					FglParameter fglParameter = this._parameterNode.Inputs[i];
					if (i > 0)
					{
						stringBuilder.Append(", ");
					}
					stringBuilder.Append(fglParameter.Name);
				}
				stringBuilder.Append(")");
				return stringBuilder.ToString();
			}
		}

		// Token: 0x06000278 RID: 632 RVA: 0x0000B1E0 File Offset: 0x000093E0
		public XElement ToXElement()
		{
			XElement xelement = new XElement("comp");
			xelement.SetAttributeValue("id", this.ID);
			xelement.SetAttributeValue("desc", this.Description);
			xelement.SetAttributeValue("for_db", this.ForDB);
			xelement.SetAttributeValue("purpose", this.Purpose);
			xelement.SetAttributeValue("status", CodeSpecStatusToString.ToString(this.Status));
			foreach (KeyValuePair<string, string> keyValuePair in this._attributes)
			{
				xelement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			if (this._parameterNode != null)
			{
				XElement xelement2 = this._parameterNode.ToXElement();
				xelement.Add(xelement2);
			}
			if (this._keywordNode != null)
			{
				XElement xelement3 = this._keywordNode.ToXElement();
				xelement.Add(xelement3);
			}
			if (this._frontCompNode != null)
			{
				XElement xelement4 = this._frontCompNode.ToXElement();
				xelement.Add(xelement4);
			}
			if (this._dimensionNode != null)
			{
				XElement xelement5 = this._dimensionNode.ToXElement();
				xelement.Add(xelement5);
			}
			if (this._testNode != null)
			{
				XElement xelement6 = this._testNode.ToXElement();
				xelement.Add(xelement6);
			}
			if (this.SpecContent != null)
			{
				XElement xelement7 = this.SpecContent.ToXElement();
				xelement.Add(xelement7);
			}
			return xelement;
		}

		// Token: 0x06000279 RID: 633 RVA: 0x0000B374 File Offset: 0x00009574
		private void OnPropertySettingChanged(string propertyName)
		{
			if ((this.Status & CodeSpecStatus.LOADED) == CodeSpecStatus.LOADED)
			{
				this.Status = CodeSpecStatus.MODIFY | this.Status;
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeSpecChangedEvent>().Publish(this.ProgramKey);
			}
			base.OnPropertyChanged(propertyName);
		}

		// Token: 0x040000DB RID: 219
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();

		// Token: 0x040000DC RID: 220
		private FglParameterNode _parameterNode;

		// Token: 0x040000DD RID: 221
		private FglKeywordNode _keywordNode;

		// Token: 0x040000DE RID: 222
		private FglDimensionNode _dimensionNode;

		// Token: 0x040000DF RID: 223
		private FglFrontCompNode _frontCompNode;

		// Token: 0x040000E0 RID: 224
		private FglTestNode _testNode;

		// Token: 0x040000E1 RID: 225
		private string _id = string.Empty;

		// Token: 0x040000E2 RID: 226
		private string _description = string.Empty;

		// Token: 0x040000E3 RID: 227
		private string _forDB = string.Empty;

		// Token: 0x040000E4 RID: 228
		private string _purpose = string.Empty;
	}
}
